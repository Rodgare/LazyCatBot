package storage

import (
	"LazyCatBot/internal/sirus"
	"testing"
)

func TestUpdate(t *testing.T) {
	players := []sirus.LeaderboardPlayer{
		{Name: "TestName1", Rank: 1, Dps: 10000},
		{Name: "TestName2", Rank: 2, Dps: 20000},
	}
	lbStorage := NewLeaderboardStorage()
	lbStorage.UpdateLeaderboardStorage(565, 0, 1, 1, players)

	if len(lbStorage.Data[565][0][1][1]) != 2 {
		t.Errorf("Expected 2 players, returned %d", len(lbStorage.Data[565][0][1][1]))
	}

}
