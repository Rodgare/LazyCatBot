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
    guild_id TEXT PRIMARY KEY,
    channel_id TEXT NOT NULL DEFAULT '',
    plan TEXT NOT NULL,
    source TEXT NOT NULL DEFAULT '',
    status TEXT NOT NULL DEFAULT 'active',
    expires_at INTEGER NOT NULL DEFAULT 0,
    created_at INTEGER NOT NULL,
    updated_at INTEGER NOT NULL
);
CREATE TABLE premium_codes (
    code TEXT PRIMARY KEY,
    guild_id TEXT NOT NULL,
    channel_id TEXT NOT NULL DEFAULT '',
    plan TEXT NOT NULL,
    created_by TEXT NOT NULL DEFAULT '',
    created_at INTEGER NOT NULL,
    expires_at INTEGER NOT NULL,
    used_at INTEGER NOT NULL DEFAULT 0,
    used_by TEXT NOT NULL DEFAULT ''
);
CREATE TABLE premium_audit (
    id INTEGER PRIMARY KEY,
    guild_id TEXT NOT NULL DEFAULT '',
    channel_id TEXT NOT NULL,
    action TEXT NOT NULL,
    plan TEXT NOT NULL DEFAULT '',
    source TEXT NOT NULL DEFAULT '',
    actor TEXT NOT NULL DEFAULT '',
    details TEXT NOT NULL DEFAULT '',
    created_at INTEGER NOT NULL
);
CREATE TABLE premium_events (
    event_key TEXT PRIMARY KEY,
    event_name TEXT NOT NULL DEFAULT '',
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
	const guild, channel = "guild-1", "chan-1"

	code, expires, err := s.CreateCode(guild, channel, "user-1", PlanMonthly)
	if err != nil {
		t.Fatalf("create code: %v", err)
	}
	if code == "" || len(code) != len("LZ-XXXX-XXXX") {
		t.Fatalf("unexpected code format %q", code)
	}
	if time.Until(expires) <= 0 {
		t.Fatalf("code expiry should be in the future")
	}
	if s.IsPremium(guild) {
		t.Fatalf("guild must not be premium before redemption")
	}

	ent, err := s.RedeemCode(code, "tg:1", false)
	if err != nil {
		t.Fatalf("redeem: %v", err)
	}
	if ent.Plan != PlanMonthly || ent.Status != "active" {
		t.Fatalf("unexpected entitlement: %+v", ent)
	}
	if !s.IsPremium(guild) {
		t.Fatalf("guild should be premium after redemption")
	}

	if _, err := s.RedeemCode(code, "tg:1", false); !errors.Is(err, ErrCodeUsed) {
		t.Fatalf("second non-recurring redeem err = %v, want ErrCodeUsed", err)
	}

	if _, err := s.RedeemCode(code, "tg:2", true); !errors.Is(err, ErrCodeSubject) {
		t.Fatalf("recurring redeem by another donor err = %v, want ErrCodeSubject", err)
	}

	before, _ := s.Get(guild)
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
	if bySubject.GuildID != guild || bySubject.ExpiresAt <= renewed.ExpiresAt {
		t.Fatalf("recurring by subject = %+v, want guild %s with later expiry", bySubject, guild)
	}
	if _, err := s.RedeemRecurringBySubject("tg:missing"); !errors.Is(err, ErrCodeNotFound) {
		t.Fatalf("unknown subject err = %v, want ErrCodeNotFound", err)
	}
}

func TestPremiumAppliesToWholeGuild(t *testing.T) {
	s := newPremiumTestStore(t)

	code, _, err := s.CreateCode("guild-42", "chan-a", "user-1", PlanMonthly)
	if err != nil {
		t.Fatalf("create code: %v", err)
	}
	if _, err := s.RedeemCode(code, "tg:1", false); err != nil {
		t.Fatalf("redeem: %v", err)
	}

	if !s.IsPremium("guild-42") {
		t.Fatalf("guild should be premium")
	}
	if s.IsPremium("guild-other") {
		t.Fatalf("other guild must stay free")
	}

	ent, err := s.Get("guild-42")
	if err != nil || ent == nil {
		t.Fatalf("Get guild entitlement = %+v, %v", ent, err)
	}
	if ent.ChannelID != "chan-a" {
		t.Fatalf("channel_id should record the requesting channel, got %q", ent.ChannelID)
	}

	// A second code for the same guild extends the same entitlement row.
	code2, _, err := s.CreateCode("guild-42", "chan-b", "user-2", PlanMonthly)
	if err != nil {
		t.Fatalf("create second code: %v", err)
	}
	extended, err := s.RedeemCode(code2, "tg:2", false)
	if err != nil {
		t.Fatalf("redeem second code: %v", err)
	}
	if extended.ExpiresAt <= ent.ExpiresAt {
		t.Fatalf("same-guild code should extend expiry: before=%d after=%d", ent.ExpiresAt, extended.ExpiresAt)
	}
	if extended.ChannelID != "chan-b" {
		t.Fatalf("channel_id should track the latest requesting channel, got %q", extended.ChannelID)
	}
}

func TestPremiumCodeExpired(t *testing.T) {
	s := newPremiumTestStore(t)
	code, _, err := s.CreateCode("guild-1", "chan-1", "user-1", PlanMonthly)
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
	const guild, channel = "guild-2", "chan-2"

	ent, err := s.Grant(guild, channel, PlanLifetime, "manual", "admin")
	if err != nil {
		t.Fatalf("grant lifetime: %v", err)
	}
	if !ent.Lifetime() || !ent.Active(time.Now()) {
		t.Fatalf("lifetime entitlement inactive: %+v", ent)
	}

	ent, err = s.Grant(guild, channel, PlanMonthly, "manual", "admin")
	if err != nil {
		t.Fatalf("grant monthly on lifetime: %v", err)
	}
	if !ent.Lifetime() {
		t.Fatalf("lifetime must not be shortened, got expires_at=%d", ent.ExpiresAt)
	}

	if err := s.Revoke(guild, "admin"); err != nil {
		t.Fatalf("revoke: %v", err)
	}
	if s.IsPremium(guild) {
		t.Fatalf("revoked guild must not be premium")
	}

	if _, err := s.Grant(guild, channel, PlanMonthly, "manual", "admin"); err != nil {
		t.Fatalf("regrant: %v", err)
	}
	if !s.IsPremium(guild) {
		t.Fatalf("regranted guild should be premium")
	}
}

func TestPremiumGrantTrialWithoutChannel(t *testing.T) {
	s := newPremiumTestStore(t)

	// channel_id is still the legacy PRIMARY KEY, so guild-wide grants must not
	// all insert the same empty string.
	for _, guild := range []string{"guild-trial-a", "guild-trial-b"} {
		ent, err := s.Grant(guild, "", PlanTrial, "trial", "admin")
		if err != nil {
			t.Fatalf("grant trial without channel for %s: %v", guild, err)
		}
		if ent.Plan != PlanTrial || !ent.Active(time.Now()) {
			t.Fatalf("unexpected trial entitlement for %s: %+v", guild, ent)
		}
		if !s.IsPremium(guild) {
			t.Fatalf("trial guild %s should be premium", guild)
		}
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

func TestClaimPremiumEvent(t *testing.T) {
	s := newPremiumTestStore(t)

	fresh, err := s.ClaimEvent("payment:1", "new_donation")
	if err != nil || !fresh {
		t.Fatalf("first claim = %v, %v; want true", fresh, err)
	}
	fresh, err = s.ClaimEvent("payment:1", "new_donation")
	if err != nil || fresh {
		t.Fatalf("duplicate claim = %v, %v; want false", fresh, err)
	}

	// Without a provider event id there is nothing to deduplicate safely.
	fresh, err = s.ClaimEvent("", "recurrent_donation")
	if err != nil || !fresh {
		t.Fatalf("empty key claim = %v, %v; want true", fresh, err)
	}
}
