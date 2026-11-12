package storage

import (
	"database/sql"
	"errors"
	"testing"
	"time"

	_ "modernc.org/sqlite"
)

const premiumTestSchema = `
CREATE TABLE premium_entitlements (
    channel_id TEXT PRIMARY KEY,
    plan TEXT NOT NULL,
    source TEXT NOT NULL DEFAULT '',
    status TEXT NOT NULL DEFAULT 'active',
    expires_at INTEGER NOT NULL DEFAULT 0,
    created_at INTEGER NOT NULL,
    updated_at INTEGER NOT NULL
);
CREATE TABLE premium_codes (
    code TEXT PRIMARY KEY,
    channel_id TEXT NOT NULL,
    plan TEXT NOT NULL,
    created_by TEXT NOT NULL DEFAULT '',
    created_at INTEGER NOT NULL,
    expires_at INTEGER NOT NULL,
    used_at INTEGER NOT NULL DEFAULT 0,
    used_by TEXT NOT NULL DEFAULT ''
);
CREATE TABLE premium_audit (
    id INTEGER PRIMARY KEY,
    channel_id TEXT NOT NULL,
    action TEXT NOT NULL,
    plan TEXT NOT NULL DEFAULT '',
    source TEXT NOT NULL DEFAULT '',
    actor TEXT NOT NULL DEFAULT '',
    details TEXT NOT NULL DEFAULT '',
    created_at INTEGER NOT NULL
);`

func newPremiumTestStore(t *testing.T) *PremiumStore {
	t.Helper()
	db, err := sql.Open("sqlite", ":memory:")
	if err != nil {
		t.Fatalf("open db: %v", err)
	}
	db.SetMaxOpenConns(1)
	t.Cleanup(func() { db.Close() })
	if _, err := db.Exec(premiumTestSchema); err != nil {
		t.Fatalf("schema: %v", err)
	}
	return NewPremiumStore(db)
}

func TestPremiumCodeLifecycle(t *testing.T) {
	s := newPremiumTestStore(t)
	const channel = "chan-1"

	code, expires, err := s.CreateCode(channel, "user-1", PlanMonthly)
	if err != nil {
		t.Fatalf("create code: %v", err)
	}
	if code == "" || len(code) != len("LZ-XXXX-XXXX") {
		t.Fatalf("unexpected code format %q", code)
	}
	if time.Until(expires) <= 0 {
		t.Fatalf("code expiry should be in the future")
	}
	if s.IsPremium(channel) {
		t.Fatalf("channel must not be premium before redemption")
	}

	ent, err := s.RedeemCode(code, "tg:1", false)
	if err != nil {
		t.Fatalf("redeem: %v", err)
	}
	if ent.Plan != PlanMonthly || ent.Status != "active" {
		t.Fatalf("unexpected entitlement: %+v", ent)
	}
	if !s.IsPremium(channel) {
		t.Fatalf("channel should be premium after redemption")
	}

	if _, err := s.RedeemCode(code, "tg:1", false); !errors.Is(err, ErrCodeUsed) {
		t.Fatalf("second non-recurring redeem err = %v, want ErrCodeUsed", err)
	}

	if _, err := s.RedeemCode(code, "tg:2", true); !errors.Is(err, ErrCodeSubject) {
		t.Fatalf("recurring redeem by another donor err = %v, want ErrCodeSubject", err)
	}

	before, _ := s.Get(channel)
	extended, err := s.RedeemCode(code, "tg:1", true)
	if err != nil {
		t.Fatalf("recurring redeem: %v", err)
	}
	if extended.ExpiresAt <= before.ExpiresAt {
		t.Fatalf("recurring redemption should extend expiry: before=%d after=%d", before.ExpiresAt, extended.ExpiresAt)
	}

	// The activation code TTL only gates the first payment. Recurring payments
	// arrive months later and must still extend the subscription.
	if _, err := s.db.Exec(`UPDATE premium_codes SET expires_at = ? WHERE code = ?`,
		time.Now().Add(-time.Hour).Unix(), code); err != nil {
		t.Fatalf("expire code after activation: %v", err)
	}
	renewed, err := s.RedeemCode(code, "tg:1", true)
	if err != nil {
		t.Fatalf("recurring after code ttl: %v", err)
	}
	if renewed.ExpiresAt <= extended.ExpiresAt {
		t.Fatalf("recurring after ttl should extend expiry: before=%d after=%d", extended.ExpiresAt, renewed.ExpiresAt)
	}

	bySubject, err := s.RedeemRecurringBySubject("tg:1")
	if err != nil {
		t.Fatalf("recurring by subject: %v", err)
	}
	if bySubject.ChannelID != channel || bySubject.ExpiresAt <= renewed.ExpiresAt {
		t.Fatalf("recurring by subject = %+v, want channel %s with later expiry", bySubject, channel)
	}
	if _, err := s.RedeemRecurringBySubject("tg:missing"); !errors.Is(err, ErrCodeNotFound) {
		t.Fatalf("unknown subject err = %v, want ErrCodeNotFound", err)
	}
}

func TestPremiumCodeExpired(t *testing.T) {
	s := newPremiumTestStore(t)
	code, _, err := s.CreateCode("chan-1", "user-1", PlanMonthly)
	if err != nil {
		t.Fatalf("create code: %v", err)
	}
	if _, err := s.db.Exec(`UPDATE premium_codes SET expires_at = ? WHERE code = ?`, time.Now().Add(-time.Hour).Unix(), code); err != nil {
		t.Fatalf("expire code: %v", err)
	}
	if _, err := s.RedeemCode(code, "tg:1", false); !errors.Is(err, ErrCodeExpired) {
		t.Fatalf("err = %v, want ErrCodeExpired", err)
	}
}

func TestPremiumGrantLifetimeAndRevoke(t *testing.T) {
	s := newPremiumTestStore(t)
	const channel = "chan-2"

	ent, err := s.Grant(channel, PlanLifetime, "manual", "admin")
	if err != nil {
		t.Fatalf("grant lifetime: %v", err)
	}
	if !ent.Lifetime() || !ent.Active(time.Now()) {
		t.Fatalf("lifetime entitlement inactive: %+v", ent)
	}

	ent, err = s.Grant(channel, PlanMonthly, "manual", "admin")
	if err != nil {
		t.Fatalf("grant monthly on lifetime: %v", err)
	}
	if !ent.Lifetime() {
		t.Fatalf("lifetime must not be shortened, got expires_at=%d", ent.ExpiresAt)
	}

	if err := s.Revoke(channel, "admin"); err != nil {
		t.Fatalf("revoke: %v", err)
	}
	if s.IsPremium(channel) {
		t.Fatalf("revoked channel must not be premium")
	}

	if _, err := s.Grant(channel, PlanMonthly, "manual", "admin"); err != nil {
		t.Fatalf("regrant: %v", err)
	}
	if !s.IsPremium(channel) {
		t.Fatalf("regranted channel should be premium")
	}
}

func TestPremiumGracePeriod(t *testing.T) {
	now := time.Now()

	cases := []struct {
		name      string
		expiresAt int64
		want      bool
	}{
		{"active", now.Add(time.Hour).Unix(), true},
		{"in grace", now.Add(-time.Hour).Unix(), true},
		{"past grace", now.Add(-PremiumGracePeriod - time.Hour).Unix(), false},
		{"lifetime", 0, true},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			e := PremiumEntitlement{Status: "active", ExpiresAt: tc.expiresAt}
			if got := e.Active(now); got != tc.want {
				t.Fatalf("Active() = %v, want %v", got, tc.want)
			}
		})
	}
}

func TestNormalizePremiumCode(t *testing.T) {
	got := NormalizePremiumCode("  lz-abcd-2345 ")
	if got != "LZ-ABCD-2345" {
		t.Fatalf("NormalizePremiumCode = %q", got)
	}
}
