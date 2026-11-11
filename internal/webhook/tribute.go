package webhook

import (
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"io"
	"log/slog"
	"net/http"
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
// registers the donor nickname typed into the donation message.
type Handler struct {
	apiKey string
	store  *storage.DonorStorage
	onName func(string)
	logger *slog.Logger
}

func NewHandler(apiKey string, store *storage.DonorStorage, onName func(string), logger *slog.Logger) *Handler {
	return &Handler{apiKey: apiKey, store: store, onName: onName, logger: logger}
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

	switch ev.Name {
	case "new_donation", "recurrent_donation":
		nick := extractDonorNick(ev.Payload.Message)
		if nick == "" {
			h.logger.Info("[Tribute] donation skipped: no nickname in message",
				"amount", ev.Payload.Amount, "currency", ev.Payload.Currency,
				"tg_user", ev.Payload.TelegramUserID, "tg_username", ev.Payload.TelegramUsername)
			break
		}
		if err := h.store.Upsert(nick, ev.Payload.Amount, ev.Payload.Currency); err != nil {
			h.logger.Error("[Tribute] save donor err", "error", err, "nick", nick)
			break
		}
		if h.onName != nil {
			h.onName(nick)
		}
		h.logger.Info("[Tribute] donor registered", "nick", nick,
			"amount", ev.Payload.Amount, "currency", ev.Payload.Currency,
			"tg_username", ev.Payload.TelegramUsername)
	default:
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

// extractDonorNick pulls a character nickname out of the donation message.
// Donors are asked to type just their nickname, but a leading "ник:" is also
// tolerated. Only the first token is used because Sirus names have no spaces.
func extractDonorNick(msg string) string {
	msg = strings.TrimSpace(msg)
	if msg == "" {
		return ""
	}

	low := strings.ToLower(msg)
	if idx := strings.Index(low, "ник:"); idx >= 0 {
		if f := strings.Fields(msg[idx+len("ник:"):]); len(f) > 0 {
			return cleanNick(f[0])
		}
	}

	f := strings.Fields(msg)
	if len(f) == 0 {
		return ""
	}
	return cleanNick(f[0])
}

func cleanNick(s string) string {
	s = strings.TrimSpace(s)
	s = strings.TrimPrefix(s, "@")
	s = strings.Trim(s, ".,;:!?()[]{}\"'«»")
	return s
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
