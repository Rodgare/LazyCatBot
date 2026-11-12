package migrations

import (
	"database/sql"
	"testing"

	"github.com/pressly/goose/v3"
	_ "modernc.org/sqlite"
)

func TestMigrationsApply(t *testing.T) {
	db, err := sql.Open("sqlite", ":memory:")
	if err != nil {
		t.Fatalf("open db: %v", err)
	}
	db.SetMaxOpenConns(1)
	defer db.Close()

	goose.SetBaseFS(EmbedFS)
	if err := goose.SetDialect("sqlite3"); err != nil {
		t.Fatalf("set dialect: %v", err)
	}
	if err := goose.Up(db, "."); err != nil {
		t.Fatalf("apply migrations: %v", err)
	}

	for _, table := range []string{
		"premium_entitlements",
		"premium_codes",
		"premium_audit",
		"premium_events",
		"leaderboard_history",
		"kill_history",
		"kill_players",
	} {
		var name string
		if err := db.QueryRow(`SELECT name FROM sqlite_master WHERE type = 'table' AND name = ?`, table).Scan(&name); err != nil {
			t.Fatalf("table %s not created: %v", table, err)
		}
	}

	// Premium is sold per Discord server, so codes and entitlements carry the
	// guild id next to the channel that requested the code.
	for table, column := range map[string]string{
		"premium_entitlements": "guild_id",
		"premium_codes":        "guild_id",
		"premium_audit":        "guild_id",
	} {
		var found int
		if err := db.QueryRow(`SELECT COUNT(*) FROM pragma_table_info(?) WHERE name = ?`, table, column).Scan(&found); err != nil {
			t.Fatalf("inspect %s.%s: %v", table, column, err)
		}
		if found != 1 {
			t.Fatalf("column %s.%s missing", table, column)
		}
	}
}
