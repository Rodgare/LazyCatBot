package storage

import (
	"database/sql"
	"io"
	"log/slog"
	"testing"
	"time"

	_ "modernc.org/sqlite"
)

func testLogger() *slog.Logger {
	return slog.New(slog.NewTextHandler(io.Discard, nil))
}

const leaderboardRangeTestSchema = `
CREATE TABLE guild_members (
    id INTEGER,
    name TEXT,
    ilvl INTEGER,
    guild_id INTEGER,
    realm TEXT DEFAULT 'x3',
    PRIMARY KEY (name, guild_id)
);
CREATE TABLE character_cache (
    realm TEXT NOT NULL,
    name TEXT NOT NULL,
    title TEXT DEFAULT '',
    mythic_rating REAL DEFAULT 0,
    black_diamonds INTEGER DEFAULT 0,
    updated_at INTEGER DEFAULT 0,
    PRIMARY KEY (realm, name)
);
CREATE TABLE leaderboard_history (
    realm TEXT NOT NULL,
    raid_id INTEGER NOT NULL,
    boss_id INTEGER NOT NULL,
    class_id INTEGER NOT NULL,
    spec_id INTEGER NOT NULL,
    player_name TEXT NOT NULL,
    ilvl INTEGER NOT NULL DEFAULT 0,
    guild_id INTEGER NOT NULL DEFAULT 0,
    zodiac INTEGER NOT NULL DEFAULT 0,
    category INTEGER NOT NULL DEFAULT 0,
    t4 INTEGER NOT NULL DEFAULT 0,
    set_pieces TEXT NOT NULL DEFAULT '',
    role TEXT NOT NULL DEFAULT '',
    dps INTEGER NOT NULL DEFAULT 0,
    hps INTEGER NOT NULL DEFAULT 0,
    snapshot_week TEXT NOT NULL,
    captured_at INTEGER NOT NULL,
    PRIMARY KEY (realm, raid_id, boss_id, class_id, spec_id, player_name, snapshot_week)
);`

func newLeaderboardTestDB(t *testing.T) *sql.DB {
	t.Helper()
	db, err := sql.Open("sqlite", ":memory:")
	if err != nil {
		t.Fatalf("open db: %v", err)
	}
	db.SetMaxOpenConns(1)
	t.Cleanup(func() { db.Close() })
	if _, err := db.Exec(leaderboardRangeTestSchema); err != nil {
		t.Fatalf("schema: %v", err)
	}
	return db
}

func TestGetBossTopRangeAggregatesHistory(t *testing.T) {
	db := newLeaderboardTestDB(t)
	store := NewLeaderboardStorage(db, testLogger())

	for _, gm := range []struct {
		name    string
		guildID int
	}{
		{"Alice", 100},
		{"Bob", 100},
		{"Carol", 200},
	} {
		if _, err := db.Exec(`INSERT INTO guild_members (name, guild_id, realm) VALUES (?, ?, 'x3')`,
			gm.name, gm.guildID); err != nil {
			t.Fatalf("insert guild member: %v", err)
		}
	}
	if _, err := db.Exec(`INSERT INTO character_cache (realm, name, title, black_diamonds) VALUES ('x3', 'Alice', 'Test', 3)`); err != nil {
		t.Fatalf("insert cache: %v", err)
	}

	now := time.Now().Unix()
	insert := func(player string, dps int, week string, capturedAt int64) {
		t.Helper()
		_, err := db.Exec(`INSERT INTO leaderboard_history
			(realm, raid_id, boss_id, class_id, spec_id, player_name, ilvl, guild_id, role, dps, snapshot_week, captured_at)
			VALUES ('x3', 1, 2, 1, 1, ?, 250, 0, 'dps', ?, ?, ?)`,
			player, dps, week, capturedAt)
		if err != nil {
			t.Fatalf("insert history: %v", err)
		}
	}

	insert("Alice", 100, "2026-10-01", now-24*3600)
	insert("Alice", 300, "2026-09-24", now-10*24*3600)
	insert("Alice", 900, "2026-09-10", now-40*24*3600)
	insert("Bob", 200, "2026-10-01", now-24*3600)
	insert("Carol", 5000, "2026-10-01", now-24*3600)

	players, err := store.GetBossTopRange("x3", 1, 2, 100, "dps", Range4W)
	if err != nil {
		t.Fatalf("4w query: %v", err)
	}
	if len(players) != 2 {
		t.Fatalf("4w players = %d, want 2: %+v", len(players), players)
	}
	if players[0].Name != "Alice" || players[0].Dps != 300 {
		t.Fatalf("4w top = %+v, want Alice 300", players[0])
	}
	if players[0].BlackDiamonds != 3 || players[0].Title != "Test" {
		t.Fatalf("character cache not joined: %+v", players[0])
	}

	players, err = store.GetBossTopRange("x3", 1, 2, 100, "dps", RangeAll)
	if err != nil {
		t.Fatalf("all query: %v", err)
	}
	if len(players) != 2 || players[0].Name != "Alice" || players[0].Dps != 900 {
		t.Fatalf("all-time top = %+v, want Alice 900", players)
	}
}
