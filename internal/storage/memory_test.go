package storage

import (
	"LazyCatBot/internal/sirus"
	"testing"
)

func TestUpdate(t *testing.T) {
	players := []sirus.LeaderboardPlayer{
		{Name: "TestName1", Rank: 1, Dps: 10000, MapID: 565, EncounterID: 0, Difficulty: 0},
		{Name: "TestName2", Rank: 2, Dps: 20000, MapID: 565, EncounterID: 0, Difficulty: 0},
	}
	lbStorage := NewLeaderboardStorage()
	lbStorage.Update(565, 0, 0, "1:1", players)

	if len(lbStorage.Data[565][0][0]["1:1"]) != 2 {
		t.Errorf("Expected 2 players, returned %d", len(lbStorage.Data[565][0][0]["1:1"]))
	}

}
