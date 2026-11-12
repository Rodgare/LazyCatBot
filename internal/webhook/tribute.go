package webhook

import (
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"regexp"
	"strings"
	"time"

	"LazyCatBot/internal/storage"
)

type tributeEvent struct {
	Name    string `json:"name"`
	Payload struct {
		Message          string `json:"message"`
		Amount           int64  `json:"amount"`
		Currency         string `json:"currency"`
		Anonymously      bool   `json:"anonymously"`
		DonationName     string `json:"donation_name"`
		TelegramUserID   int64  `json:"telegram_user_id"`
		TelegramUsername string `json:"telegram_username"`
	} `json:"payload"`
}

// Handler receives Tribute webhooks, verifies the trbt-signature HMAC and
// activates or extends a channel's premium subscription from the code in the
// payment message.
type Handler struct {
	apiKey  string
	premium *storage.PremiumStore
	logger  *slog.Logger
}

func NewHandler(apiKey string, premium *storage.PremiumStore, logger *slog.Logger) *Handler {
	return &Handler{apiKey: apiKey, premium: premium, logger: logger}
}

func (h *Handler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		w.WriteHeader(http.StatusMethodNotAllowed)
		return
	}

	body, err := io.ReadAll(io.LimitReader(r.Body, 1<<20))
	if err != nil {
		http.Error(w, "read body", http.StatusBadRequest)
		return
	}

	sig := r.Header.Get("trbt-signature")
	if sig == "" {
		sig = r.Header.Get("Trbt-Signature")
	}
	if !verifySignature(body, sig, h.apiKey) {
		h.logger.Warn("[Tribute] invalid webhook signature", "remote", r.RemoteAddr)
		w.WriteHeader(http.StatusUnauthorized)
		return
	}

	var ev tributeEvent
	if err := json.Unmarshal(body, &ev); err != nil {
		http.Error(w, "bad json", http.StatusBadRequest)
		return
	}

	if isPaymentEvent(ev.Name) {
		h.handlePayment(ev)
	} else {
		h.logger.Info("[Tribute] webhook ignored", "name", ev.Name)
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	_, _ = w.Write([]byte(`{"status":"ok"}`))
}

// verifySignature checks the HMAC-SHA256 of the raw body against the
// trbt-signature header, signed with the Tribute API key. Hex and base64
// encodings are both accepted.
func verifySignature(body []byte, sig, key string) bool {
	if key == "" || sig == "" {
		return false
	}
	mac := hmac.New(sha256.New, []byte(key))
	mac.Write(body)
	sum := mac.Sum(nil)

	sig = strings.TrimSpace(sig)

	expectedHex := hex.EncodeToString(sum)
	if hmac.Equal([]byte(expectedHex), []byte(strings.ToLower(sig))) {
		return true
	}
	if hmac.Equal([]byte(base64.StdEncoding.EncodeToString(sum)), []byte(sig)) {
		return true
	}
	return false
}

var premiumCodeRe = regexp.MustCompile(`(?i)\bLZ-[A-Z0-9]{4}-[A-Z0-9]{4}\b`)

// extractPremiumCode finds an activation code in the payment message and
// returns the code plus the message with the code removed.
func extractPremiumCode(msg string) (string, string) {
	loc := premiumCodeRe.FindStringIndex(msg)
	if loc == nil {
		return "", msg
	}
	code := storage.NormalizePremiumCode(msg[loc[0]:loc[1]])
	rest := strings.TrimSpace(msg[:loc[0]] + " " + msg[loc[1]:])
	return code, rest
}

func tributeSubject(ev tributeEvent) string {
	if ev.Payload.TelegramUserID > 0 {
		return fmt.Sprintf("tg:%d", ev.Payload.TelegramUserID)
	}
	if u := strings.ToLower(strings.TrimSpace(ev.Payload.TelegramUsername)); u != "" {
		return "tgname:" + u
	}
	return ""
}

func isPaymentEvent(name string) bool {
	switch name {
	case "new_donation", "new_subscription":
		return true
	case "recurrent_donation", "recurrent_subscription", "subscription_renewed":
		return true
	}
	return false
}

func isRecurringEvent(name string) bool {
	switch name {
	case "recurrent_donation", "recurrent_subscription", "subscription_renewed":
		return true
	}
	return false
}

// handlePayment activates premium from the code in the payment message, or
// renews the donor's most recent subscription when Tribute does not repeat the
// original message on a recurring payment.
func (h *Handler) handlePayment(ev tributeEvent) {
	code, _ := extractPremiumCode(ev.Payload.Message)
	if code != "" {
		h.redeemPremium(ev, code)
		return
	}

	if isRecurringEvent(ev.Name) && h.premium != nil {
		ent, err := h.premium.RedeemRecurringBySubject(tributeSubject(ev))
		if err != nil {
			h.logger.Warn("[Tribute] recurring payment not matched",
				"error", err, "tg_user", ev.Payload.TelegramUserID)
			return
		}
		h.logger.Info("[Tribute] premium renewed by subject",
			"channel_id", ent.ChannelID, "plan", ent.Plan,
			"expires_at", ent.ExpiresAt, "tg_user", ev.Payload.TelegramUserID)
		return
	}

	h.logger.Info("[Tribute] payment ignored: no premium code in message",
		"amount", ev.Payload.Amount, "currency", ev.Payload.Currency,
		"tg_user", ev.Payload.TelegramUserID, "tg_username", ev.Payload.TelegramUsername)
}

func (h *Handler) redeemPremium(ev tributeEvent, code string) {
	if h.premium == nil {
		return
	}
	recurring := isRecurringEvent(ev.Name)
	ent, err := h.premium.RedeemCode(code, tributeSubject(ev), recurring)
	if err != nil {
		h.logger.Warn("[Tribute] premium code not redeemed", "error", err, "code", code,
			"recurring", recurring, "tg_user", ev.Payload.TelegramUserID)
		return
	}
	h.logger.Info("[Tribute] premium activated",
		"channel_id", ent.ChannelID, "plan", ent.Plan, "expires_at", ent.ExpiresAt,
		"recurring", recurring, "tg_user", ev.Payload.TelegramUserID)
}

// Start serves the webhook on addr+path until ctx is cancelled.
func Start(ctx context.Context, addr, path string, handler http.Handler, logger *slog.Logger) {
	mux := http.NewServeMux()
	mux.Handle(path, handler)

	srv := &http.Server{
		Addr:              addr,
		Handler:           mux,
		ReadHeaderTimeout: 5 * time.Second,
	}

	go func() {
		<-ctx.Done()
		shutdownCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		_ = srv.Shutdown(shutdownCtx)
	}()

	logger.Info("[Tribute] webhook listening", "addr", addr, "path", path)
	if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
		logger.Error("[Tribute] webhook server error", "error", err)
	}
}
