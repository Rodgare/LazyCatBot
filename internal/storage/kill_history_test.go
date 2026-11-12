package storage

import (
	"database/sql"
	"testing"

	"LazyCatBot/internal/models"

	_ "modernc.org/sqlite"
)

const killHistoryTestSchema = `
CREATE TABLE kill_history (
    realm TEXT NOT NULL,
    kill_id INTEGER NOT NULL,
    guild_id INTEGER NOT NULL,
    raid_id INTEGER NOT NULL,
    boss_id INTEGER NOT NULL,
    kills_at INTEGER NOT NULL,
    duration TEXT NOT NULL DEFAULT '',
    attempts INTEGER NOT NULL DEFAULT 0,
    total_dps INTEGER NOT NULL DEFAULT 0,
    total_hps INTEGER NOT NULL DEFAULT 0,
    created_at INTEGER NOT NULL,
    PRIMARY KEY (realm, kill_id, guild_id)
);
CREATE TABLE kill_players (
    realm TEXT NOT NULL,
    kill_id INTEGER NOT NULL,
    guild_id INTEGER NOT NULL,
    player_name TEXT NOT NULL,
    guid INTEGER NOT NULL DEFAULT 0,
    class_id INTEGER NOT NULL DEFAULT 0,
    spec_id INTEGER NOT NULL DEFAULT 0,
    role TEXT NOT NULL DEFAULT '',
    dps INTEGER NOT NULL DEFAULT 0,
    hps INTEGER NOT NULL DEFAULT 0,
    ilvl INTEGER NOT NULL DEFAULT 0,
    t4 INTEGER NOT NULL DEFAULT 0,
    set_pieces TEXT NOT NULL DEFAULT '',
    PRIMARY KEY (realm, kill_id, guild_id, player_name)
);
CREATE TABLE character_cache (
    realm TEXT NOT NULL,
    name TEXT NOT NULL,
    title TEXT DEFAULT '',
    mythic_rating REAL DEFAULT 0,
    black_diamonds INTEGER DEFAULT 0,
    updated_at INTEGER DEFAULT 0,
    PRIMARY KEY (realm, name)
);`

func newKillHistoryTestStore(t *testing.T) *KillHistoryStorage {
	t.Helper()
	db, err := sql.Open("sqlite", ":memory:")
	if err != nil {
		t.Fatalf("open db: %v", err)
	}
	db.SetMaxOpenConns(1)
	t.Cleanup(func() { db.Close() })
	if _, err := db.Exec(killHistoryTestSchema); err != nil {
		t.Fatalf("schema: %v", err)
	}
	return NewKillHistoryStorage(db, testLogger())
}

func killReport(killID, guildID int, killedAt string, players ...models.PlayerReport) models.BossKillReport {
	return models.BossKillReport{
		Realm:     "x3",
		KillID:    killID,
		GuildID:   guildID,
		RaidOrder: 631,
		BossID:    killID,
		KilledAt:  killedAt,
		Players:   players,
	}
}

func TestKillHistoryDigestAndProfile(t *testing.T) {
	store := newKillHistoryTestStore(t)

	if _, err := store.db.Exec(`INSERT INTO character_cache (realm, name, title, black_diamonds)
		VALUES ('x3', 'Alice', 'Тест', 4)`); err != nil {
		t.Fatalf("insert cache: %v", err)
	}

	alice1 := models.PlayerReport{Name: "Alice", GUID: 1, ClassID: 8, SpecID: 1, Ilvl: 250, Dps: 1000, Role: 0}
	bob1 := models.PlayerReport{Name: "Bob", GUID: 2, ClassID: 5, SpecID: 2, Ilvl: 255, Hps: 800, Role: 1}
	alice2 := models.PlayerReport{Name: "Alice", GUID: 1, ClassID: 8, SpecID: 1, Ilvl: 260, Dps: 1200, Role: 0}
	bob2 := models.PlayerReport{Name: "Bob", GUID: 2, ClassID: 5, SpecID: 2, Ilvl: 255, Hps: 900, Role: 1}
	other := models.PlayerReport{Name: "Alice", GUID: 1, ClassID: 8, SpecID: 1, Ilvl: 260, Dps: 9999, Role: 0}

	if err := store.RecordKill(killReport(1, 100, "2026-11-10 20:00:00", alice1, bob1)); err != nil {
		t.Fatalf("record kill 1: %v", err)
	}
	if err := store.RecordKill(killReport(2, 100, "2026-11-11 20:00:00", alice2, bob2)); err != nil {
		t.Fatalf("record kill 2: %v", err)
	}
	if err := store.RecordKill(killReport(3, 200, "2026-11-11 21:00:00", other)); err != nil {
		t.Fatalf("record other guild kill: %v", err)
	}

	since := parseKillTime("2026-11-01 00:00:00")
	weekSince := parseKillTime("2026-11-05 00:00:00")
	until := parseKillTime("2026-11-12 00:00:00")

	total, attendance, err := store.GetAttendance("x3", 100, since)
	if err != nil {
		t.Fatalf("attendance: %v", err)
	}
	if total != 2 {
		t.Fatalf("total kills = %d, want 2", total)
	}
	if len(attendance) != 2 || attendance[0].Name != "Alice" || attendance[0].Kills != 2 {
		t.Fatalf("attendance = %+v", attendance)
	}
	if attendance[0].BlackDiamonds != 4 || attendance[0].Title != "Тест" {
		t.Fatalf("attendance cache = %+v", attendance[0])
	}

	digest, err := store.GetWeeklyDigest("x3", 100, weekSince, until)
	if err != nil {
		t.Fatalf("digest: %v", err)
	}
	if digest.TotalKills != 2 {
		t.Fatalf("digest kills = %d, want 2", digest.TotalKills)
	}
	if len(digest.FirstKills) != 2 {
		t.Fatalf("first kills = %+v, want 2", digest.FirstKills)
	}
	if len(digest.TopDps) != 1 || digest.TopDps[0].Name != "Alice" || digest.TopDps[0].Value != 1200 {
		t.Fatalf("top dps = %+v", digest.TopDps)
	}
	if digest.TopDps[0].BlackDiamonds != 4 || digest.TopDps[0].Title != "Тест" {
		t.Fatalf("top dps cache = %+v", digest.TopDps[0])
	}
	if len(digest.TopHps) != 1 || digest.TopHps[0].Name != "Bob" || digest.TopHps[0].Value != 900 {
		t.Fatalf("top hps = %+v", digest.TopHps)
	}

	profile, err := store.GetPlayerProfile("x3", 100, "Alice", since)
	if err != nil {
		t.Fatalf("profile: %v", err)
	}
	if profile.TotalKills != 2 || profile.BestDps != 1200 || profile.AttendedPct != 100 {
		t.Fatalf("profile = %+v", profile)
	}
	if profile.ClassID != 8 || profile.Role != "dps" || profile.Ilvl != 260 {
		t.Fatalf("profile metadata = %+v", profile)
	}

	if err := store.RecordKill(killReport(1, 100, "2026-11-10 20:00:00", alice1, bob1)); err != nil {
		t.Fatalf("record kill 1 again: %v", err)
	}
	total, _, err = store.GetAttendance("x3", 100, since)
	if err != nil || total != 2 {
		t.Fatalf("idempotency: total=%d err=%v, want 2", total, err)
	}
}
