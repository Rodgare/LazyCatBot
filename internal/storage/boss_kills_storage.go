package storage

import (
	"LazyCatBot/internal/sirus"
	"sync"
)

type BossKillsStorage struct {
	mu sync.RWMutex
	// [KillID]
	Kills map[int]bool
}

func NewBossKillsStorage() *BossKillsStorage {
	return &BossKillsStorage{
		Kills: make(map[int]bool),
	}
}

func (s *BossKillsStorage) UpdateBossKillsStorage(kills []sirus.BossKill) {
	s.mu.Lock()
	defer s.mu.Unlock()

	if s.Kills == nil {
		s.Kills = make(map[int]bool)
	}

	for _, kill := range kills {
		s.Kills[kill.KillID] = true
	}
}

func (s *BossKillsStorage) IsNew(killID int) bool {
	s.mu.RLock()
	defer s.mu.RUnlock()

	return !s.Kills[killID]
}
