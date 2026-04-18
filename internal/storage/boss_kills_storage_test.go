package storage

import (
	"LazyCatBot/internal/sirus"
	"testing"
)

func TestUpdateBossKillsStorage(t *testing.T) {
	mock := []sirus.BossKill{
		{KillID: 12345, GuildId: 222, TimeEnd: "2026-04-17 23:35:00"},
	}

	store := NewBossKillsStorage()
	store.UpdateBossKillsStorage(mock)

	if !store.Kills[12345] {
		t.Errorf("Expected KillID 12345 to be stored, but it wasn't")
	}

	if store.IsNew(12345) == true {
		t.Errorf("Expected IsNew(12345) to be false, but got true")
	}
}
