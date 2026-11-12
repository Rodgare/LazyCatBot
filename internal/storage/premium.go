package storage

import (
	"crypto/rand"
	"database/sql"
	"errors"
	"fmt"
	"math/big"
	"strings"
	"time"

	_ "modernc.org/sqlite"
)

// PremiumPlan identifies a paid plan. Durations are intentionally expressed in
// days: monthly/yearly are billing periods, not exact calendar months, and
// lifetime is stored as expires_at = 0.
type PremiumPlan string

const (
	PlanMonthly  PremiumPlan = "monthly"
	PlanYearly   PremiumPlan = "yearly"
	PlanLifetime PremiumPlan = "lifetime"
	PlanTrial    PremiumPlan = "trial"
)

// PremiumGracePeriod is how long premium features stay active after expires_at,
// so a slightly late renewal does not immediately cut access.
const PremiumGracePeriod = 72 * time.Hour

const premiumCodeTTL = 24 * time.Hour

var (
	ErrInvalidPlan       = errors.New("invalid premium plan")
	ErrCodeNotFound      = errors.New("premium code not found")
	ErrCodeUsed          = errors.New("premium code already used")
	ErrCodeExpired       = errors.New("premium code expired")
	ErrCodeSubject       = errors.New("premium code belongs to another donor")
	ErrEntitlementAbsent = errors.New("premium entitlement not found")
)

func (p PremiumPlan) Valid() bool {
	switch p {
	case PlanMonthly, PlanYearly, PlanLifetime, PlanTrial:
		return true
	}
	return false
}

// Duration returns the subscription length. Lifetime returns 0, which is the
// sentinel used by the storage layer for "never expires".
func (p PremiumPlan) Duration() time.Duration {
	switch p {
	case PlanMonthly:
		return 31 * 24 * time.Hour
	case PlanYearly:
		return 366 * 24 * time.Hour
	case PlanTrial:
		return 8 * 24 * time.Hour
	case PlanLifetime:
		return 0
	}
	return 0
}

func (p PremiumPlan) Title() string {
	switch p {
	case PlanMonthly:
		return "1 месяц"
	case PlanYearly:
		return "12 месяцев"
	case PlanLifetime:
		return "Навсегда"
	case PlanTrial:
		return "Пробный период"
	}
	return string(p)
}

// PremiumEntitlement is a paid subscription. Premium is sold per Discord
// server: the code is created in one channel, but it activates premium for the
// whole guild, so every channel of that server shares the same subscription.
// ChannelID records where the code was requested and is kept for audit only.
type PremiumEntitlement struct {
	GuildID   string
	ChannelID string
	Plan      PremiumPlan
	Source    string
	Status    string
	ExpiresAt int64
	CreatedAt int64
	UpdatedAt int64
}

// Lifetime reports whether the entitlement never expires.
func (e PremiumEntitlement) Lifetime() bool {
	return e.ExpiresAt == 0
}

// Active reports whether premium features are available right now, including
// the grace period after expiry.
func (e PremiumEntitlement) Active(now time.Time) bool {
	if e.Status != "active" {
		return false
	}
	if e.Lifetime() {
		return true
	}
	return now.Unix() <= e.ExpiresAt+int64(PremiumGracePeriod/time.Second)
}

// ExpiresAtTime returns the expiry as time.Time. For lifetime plans the zero
// time is returned; check Lifetime() first.
func (e PremiumEntitlement) ExpiresAtTime() time.Time {
	if e.ExpiresAt == 0 {
		return time.Time{}
	}
	return time.Unix(e.ExpiresAt, 0)
}

type PremiumStore struct {
	db *sql.DB

	// legacyMode keeps the old per-channel lookup working for rows created
	// before premium became guild-scoped.
	legacyMode bool
}

func NewPremiumStore(db *sql.DB) *PremiumStore {
	return &PremiumStore{db: db}
}

// EnableLegacyPerChannel switches lookups without a guild id to the old
// per-channel behaviour. It exists for tests written before guild scoping.
func (s *PremiumStore) EnableLegacyPerChannel() {
	s.legacyMode = true
}

// IsPremium is the hot-path check used by workers and handlers. Premium is
// guild-wide: any channel of the server is premium when the server is. Errors
// are treated as "not premium" so a broken DB never grants access.
func (s *PremiumStore) IsPremium(guildID string) bool {
	e, err := s.Get(guildID)
	if err != nil || e == nil {
		return false
	}
	return e.Active(time.Now())
}

// Get returns the entitlement for a server, or nil when none exists. An empty
// guild id falls back to a legacy per-channel lookup in legacyMode.
func (s *PremiumStore) Get(guildID string) (*PremiumEntitlement, error) {
	if guildID == "" {
		return nil, nil
	}
	const cols = `SELECT guild_id, channel_id, plan, source, status, expires_at, created_at, updated_at`
	query := cols + ` FROM premium_entitlements WHERE guild_id = ?`
	args := []any{guildID}
	if s.legacyMode {
		query = cols + ` FROM premium_entitlements WHERE channel_id = ?`
	}

	var e PremiumEntitlement
	err := s.db.QueryRow(query, args...).Scan(
		&e.GuildID, &e.ChannelID, &e.Plan, &e.Source, &e.Status,
		&e.ExpiresAt, &e.CreatedAt, &e.UpdatedAt)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, nil
		}
		return nil, err
	}
	return &e, nil
}

// Grant activates or extends premium for a server. A new period starts from
// the current expiry when the existing entitlement is still active, so renewals
// never lose remaining time. Granting anything on an active lifetime plan keeps
// it lifetime. Every change is written to premium_audit.
func (s *PremiumStore) Grant(guildID, channelID string, plan PremiumPlan, source, actor string) (*PremiumEntitlement, error) {
	if guildID == "" {
		return nil, fmt.Errorf("empty guild id")
	}
	if !plan.Valid() {
		return nil, ErrInvalidPlan
	}

	tx, err := s.db.Begin()
	if err != nil {
		return nil, err
	}
	defer tx.Rollback()

	e, err := grantTx(tx, guildID, channelID, plan, source, actor, "grant")
	if err != nil {
		return nil, err
	}
	if err := tx.Commit(); err != nil {
		return nil, err
	}
	return e, nil
}

// Revoke disables premium for a server immediately. The row is kept for audit
// history; a later Grant reactivates it.
func (s *PremiumStore) Revoke(guildID, actor string) error {
	if guildID == "" {
		return fmt.Errorf("empty guild id")
	}
	now := time.Now().Unix()
	res, err := s.db.Exec(`UPDATE premium_entitlements SET status = 'revoked', updated_at = ? WHERE guild_id = ?`,
		now, guildID)
	if err != nil {
		return err
	}
	affected, _ := res.RowsAffected()
	if affected == 0 {
		return ErrEntitlementAbsent
	}
	return s.audit(guildID, "revoke", "", "manual", actor, "")
}

// ClaimEvent records a payment webhook event key and reports whether this is
// the first delivery. Empty keys are always accepted: without a provider event
// id there is nothing safe to deduplicate on, and a hash of amount+user would
// wrongly drop legitimate monthly renewals.
func (s *PremiumStore) ClaimEvent(key, name string) (bool, error) {
	if key == "" {
		return true, nil
	}
	res, err := s.db.Exec(`INSERT OR IGNORE INTO premium_events (event_key, event_name, created_at)
		VALUES (?, ?, ?)`, key, name, time.Now().Unix())
	if err != nil {
		return false, err
	}
	affected, err := res.RowsAffected()
	if err != nil {
		return false, err
	}
	return affected == 1, nil
}

// CreateCode generates a single-use activation code. Codes are short enough to
// type into a donation message and valid for 24 hours. The code activates
// premium for the whole Discord server; channelID only records where the owner
// requested it.
func (s *PremiumStore) CreateCode(guildID, channelID, createdBy string, plan PremiumPlan) (string, time.Time, error) {
	if guildID == "" {
		return "", time.Time{}, fmt.Errorf("empty guild id")
	}
	if !plan.Valid() {
		return "", time.Time{}, ErrInvalidPlan
	}

	now := time.Now()
	expires := now.Add(premiumCodeTTL)

	for attempt := 0; attempt < 5; attempt++ {
		code, err := generatePremiumCode()
		if err != nil {
			return "", time.Time{}, err
		}
		_, err = s.db.Exec(`INSERT INTO premium_codes
			(code, guild_id, channel_id, plan, created_by, created_at, expires_at, used_at, used_by)
			VALUES (?, ?, ?, ?, ?, ?, ?, 0, '')`,
			code, guildID, channelID, string(plan), createdBy, now.Unix(), expires.Unix())
		if err == nil {
			return code, expires, nil
		}
		if !isUniqueConstraint(err) {
			return "", time.Time{}, err
		}
	}
	return "", time.Time{}, fmt.Errorf("could not generate unique premium code")
}

// RedeemCode activates the plan encoded in a code. New donations require a
// fresh code. Recurrent donations may reuse the code they were first redeemed
// with, but only by the same donor subject.
func (s *PremiumStore) RedeemCode(code, subject string, recurring bool) (*PremiumEntitlement, error) {
	code = NormalizePremiumCode(code)
	if code == "" {
		return nil, ErrCodeNotFound
	}

	tx, err := s.db.Begin()
	if err != nil {
		return nil, err
	}
	defer tx.Rollback()

	var (
		guildID   string
		channelID string
		plan      PremiumPlan
		expiresAt int64
		usedAt    int64
		usedBy    string
	)
	err = tx.QueryRow(`SELECT guild_id, channel_id, plan, expires_at, used_at, used_by
		FROM premium_codes WHERE code = ?`, code).
		Scan(&guildID, &channelID, &plan, &expiresAt, &usedAt, &usedBy)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, ErrCodeNotFound
	}
	if err != nil {
		return nil, err
	}

	now := time.Now()
	if !recurring && now.Unix() > expiresAt {
		return nil, ErrCodeExpired
	}

	if recurring {
		if usedAt == 0 {
			return nil, ErrCodeUsed
		}
		if usedBy != "" && subject != "" && usedBy != subject {
			return nil, ErrCodeSubject
		}
	} else {
		if usedAt != 0 {
			return nil, ErrCodeUsed
		}
		if _, err := tx.Exec(`UPDATE premium_codes SET used_at = ?, used_by = ? WHERE code = ?`,
			now.Unix(), subject, code); err != nil {
			return nil, err
		}
	}

	e, err := grantTx(tx, guildID, channelID, plan, "tribute", subject, "redeem")
	if err != nil {
		return nil, err
	}
	if _, err := tx.Exec(`INSERT INTO premium_audit
		(guild_id, action, plan, source, actor, details, created_at)
		VALUES (?, ?, ?, ?, ?, ?, ?)`,
		guildID, "redeem", string(plan), "tribute", subject, code, now.Unix()); err != nil {
		return nil, err
	}
	if err := tx.Commit(); err != nil {
		return nil, err
	}
	return e, nil
}

// RedeemRecurringBySubject extends the most recently activated subscription
// for a donor. It is the fallback for Tribute recurring events that do not
// repeat the original message with the activation code.
func (s *PremiumStore) RedeemRecurringBySubject(subject string) (*PremiumEntitlement, error) {
	if subject == "" {
		return nil, ErrCodeNotFound
	}
	var code string
	err := s.db.QueryRow(`SELECT code FROM premium_codes
		WHERE used_by = ? ORDER BY used_at DESC LIMIT 1`, subject).Scan(&code)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, ErrCodeNotFound
	}
	if err != nil {
		return nil, err
	}
	return s.RedeemCode(code, subject, true)
}

func grantTx(tx *sql.Tx, guildID, channelID string, plan PremiumPlan, source, actor, action string) (*PremiumEntitlement, error) {
	if !plan.Valid() {
		return nil, ErrInvalidPlan
	}

	now := time.Now()
	nowUnix := now.Unix()

	var (
		currentExpiresAt int64
		currentStatus    string
		currentCreatedAt int64
	)
	err := tx.QueryRow(`SELECT expires_at, status, created_at FROM premium_entitlements WHERE guild_id = ?`, guildID).
		Scan(&currentExpiresAt, &currentStatus, &currentCreatedAt)
	exists := true
	if errors.Is(err, sql.ErrNoRows) {
		exists = false
	} else if err != nil {
		return nil, err
	}

	var newExpiresAt int64
	switch {
	case plan == PlanLifetime:
		newExpiresAt = 0
	case exists && currentStatus == "active" && currentExpiresAt == 0:
		// Already lifetime: never shorten it.
		newExpiresAt = 0
	default:
		base := nowUnix
		if exists && currentStatus == "active" && currentExpiresAt > base {
			base = currentExpiresAt
		}
		newExpiresAt = base + int64(plan.Duration()/time.Second)
	}

	createdAt := nowUnix
	if exists {
		createdAt = currentCreatedAt
		if _, err := tx.Exec(`UPDATE premium_entitlements
			SET channel_id = ?, plan = ?, source = ?, status = 'active', expires_at = ?, updated_at = ?
			WHERE guild_id = ?`,
			channelID, string(plan), source, newExpiresAt, nowUnix, guildID); err != nil {
			return nil, err
		}
	} else {
		if _, err := tx.Exec(`INSERT INTO premium_entitlements
			(guild_id, channel_id, plan, source, status, expires_at, created_at, updated_at)
			VALUES (?, ?, ?, ?, 'active', ?, ?, ?)`,
			guildID, channelID, string(plan), source, newExpiresAt, createdAt, nowUnix); err != nil {
			return nil, err
		}
	}

	if _, err := tx.Exec(`INSERT INTO premium_audit
		(guild_id, action, plan, source, actor, details, created_at)
		VALUES (?, ?, ?, ?, ?, ?, ?)`,
		guildID, action, string(plan), source, actor, channelID, nowUnix); err != nil {
		return nil, err
	}

	return &PremiumEntitlement{
		GuildID:   guildID,
		ChannelID: channelID,
		Plan:      plan,
		Source:    source,
		Status:    "active",
		ExpiresAt: newExpiresAt,
		CreatedAt: createdAt,
		UpdatedAt: nowUnix,
	}, nil
}

func (s *PremiumStore) audit(guildID, action, plan, source, actor, details string) error {
	_, err := s.db.Exec(`INSERT INTO premium_audit
		(guild_id, action, plan, source, actor, details, created_at)
		VALUES (?, ?, ?, ?, ?, ?, ?)`,
		guildID, action, plan, source, actor, details, time.Now().Unix())
	return err
}

const premiumCodeAlphabet = "23456789ABCDEFGHJKLMNPQRSTUVWXYZ"

func generatePremiumCode() (string, error) {
	const groups, groupLen = 2, 4
	buf := make([]byte, groups*groupLen)
	limit := big.NewInt(int64(len(premiumCodeAlphabet)))
	for i := range buf {
		n, err := rand.Int(rand.Reader, limit)
		if err != nil {
			return "", err
		}
		buf[i] = premiumCodeAlphabet[n.Int64()]
	}
	parts := make([]string, 0, groups)
	for i := 0; i < groups; i++ {
		parts = append(parts, string(buf[i*groupLen:(i+1)*groupLen]))
	}
	return "LZ-" + strings.Join(parts, "-"), nil
}

// NormalizePremiumCode trims and uppercases a user-supplied code.
func NormalizePremiumCode(code string) string {
	return strings.ToUpper(strings.TrimSpace(code))
}

func isUniqueConstraint(err error) bool {
	if err == nil {
		return false
	}
	msg := strings.ToLower(err.Error())
	return strings.Contains(msg, "unique constraint") || strings.Contains(msg, "constraint failed")
}
