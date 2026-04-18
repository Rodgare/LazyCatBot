package storage

import (
	"LazyCatBot/internal/sirus"
	"testing"
)

func TestUpdate(t *testing.T) {
	players := []sirus.LeaderboardPlayer{
		{Name: "TestName1", Rank: 1, Dps: 20000},
		{Name: "TestName2", Rank: 2, Dps: 10000},
	}
	lbStorage := NewLeaderboardStorage()
	lbStorage.UpdateLeaderboardStorage(565, 0, 1, 1, players)

	if len(lbStorage.Data[565][0][1][1]) != 2 {
		t.Errorf("Expected 2 players, returned %d", len(lbStorage.Data[565][0][1][1]))
	}

}

func TestGetSpecRank(t *testing.T) {
	players := []sirus.LeaderboardPlayer{
		{Name: "TestName1", Rank: 1, Dps: 50000},
		{Name: "TestName2", Rank: 2, Dps: 40000},
		{Name: "TestName3", Rank: 3, Dps: 30000},
		{Name: "TestName4", Rank: 4, Dps: 20000},
		{Name: "TestName5", Rank: 5, Dps: 10000},
	}
	lbStorage := NewLeaderboardStorage()
	lbStorage.UpdateLeaderboardStorage(11, 0, 1, 0, players)
	rank := lbStorage.GetSpecRank(11, 0, 1, 0, 35000)

	if rank != 3 {
		t.Errorf("Expected rank 3, returned %d", rank)
	}

	rank = lbStorage.GetSpecRank(0, 0, 0, 0, 35000)
	if rank != 0 {
		t.Error("Expected rank 0 for undefined key")
	}
}
