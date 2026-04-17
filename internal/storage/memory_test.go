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
	lbStorage.Update(1, 0, "1:1", players)

	if len(lbStorage.Data[1][0]["1:1"]) != 2 {
		t.Errorf("Expected 2 players, returned %d", len(lbStorage.Data[1][0]["1:1"]))
	}

	lbStorage.Update(1, 0, "1:1", players)
	if len(lbStorage.Data[1][0]["1:1"]) != 2 {
		t.Errorf("Expected 2 players, returned %d", len(lbStorage.Data[1][0]["1:1"]))
	}

	players2 := []sirus.LeaderboardPlayer{
		{Name: "TestName3", Rank: 1, Dps: 4444},
		{Name: "TestName4", Rank: 2, Dps: 3232},
	}
	lbStorage.Update(1, 0, "1:1", players2)
	if len(lbStorage.Data[1][0]["1:1"]) != 2 {
		t.Errorf("Expected 2 players, returned %d", len(lbStorage.Data[1][0]["1:1"]))
	}

	lbStorage.Update(2, 1, "11:2", players2)
	if len(lbStorage.Data[1][0]["1:1"]) != 2 {
		t.Errorf("Expected 2 players, returned %d", len(lbStorage.Data[1][0]["1:1"]))
	}

	if lbStorage.Data[2][1]["11:2"][0].Name != "TestName3" {
		t.Errorf("Expected player name TestName3, returned %s", lbStorage.Data[2][1]["11:2"][0].Name)
	}
}
