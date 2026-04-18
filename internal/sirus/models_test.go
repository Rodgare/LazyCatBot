package sirus

import (
	"encoding/json"
	"os"
	"testing"
)

func TestLeaderboard(t *testing.T) {
	data, err := os.ReadFile("testdata/pve_example.json")
	if err != nil {
		t.Fatalf("File reading error %v", err)
	}
	var lb Leaderboard
	if err := json.Unmarshal(data, &lb); err != nil {
		t.Errorf("Json parsing error %v", err)
	}
	if len(lb.Data) == 0 {
		t.Error("Empty json data")
	}
	if lb.Data[0].Name != "Aibolit" {
		t.Errorf("expected name Aibolit, but got %s", lb.Data[0].Name)
	}
}

func TestLatestBossKills(t *testing.T) {
	data, err := os.ReadFile("testdata/guildkills_example.json")
	if err != nil {
		t.Fatalf("File reading error %v", err)
	}
	var gk LatestBossKills
	if err := json.Unmarshal(data, &gk); err != nil {
		t.Errorf("Json parsing error %v", err)
	}
	if len(gk.Data) == 0 {
		t.Error("Empty json data")
	}
	if gk.Data[0].KillID != 8244752 {
		t.Errorf("expected id 8244752, but got %d", gk.Data[0].KillID)
	}
}
