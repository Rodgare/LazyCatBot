package storage

import (
	"LazyCatBot/internal/sirus"
	"database/sql"
	"testing"
)

func TestLeaderboardSQL(t *testing.T) {
	db, err := sql.Open("sqlite", ":memory:")
	if err != nil {
		t.Fatalf("Failed to open memory db: %v", err)
	}

	lbStorage := NewLeaderboardStorage(db)

	defer db.Close()

	if err := lbStorage.InitDB(); err != nil {
		t.Fatalf("Failed to init db: %v", err)
	}

	players := []sirus.LeaderboardPlayer{
		{Name: "ProPlayer", ClassID: 1, SpecID: 0, Dps: 50000},
		{Name: "NoobPlayer", ClassID: 1, SpecID: 0, Dps: 10000},
	}

	err = lbStorage.UpdateLeaderboardStorage(11, 0, players)
	if err != nil {
		t.Fatalf("Update failed: %v", err)
	}

	rank1 := lbStorage.GetSpecRank(11, 0, 1, 0, 50000)
	if rank1 != 1 {
		t.Errorf("Expected rank 1, got %d", rank1)
	}

	rank2 := lbStorage.GetSpecRank(11, 0, 1, 0, 30000)
	if rank2 != 2 {
		t.Errorf("Expected rank 2, got %d", rank2)
	}

	rank3 := lbStorage.GetSpecRank(11, 0, 1, 0, 5000)
	if rank3 != 3 {
		t.Errorf("Expected rank 3, got %d", rank3)
	}
}
