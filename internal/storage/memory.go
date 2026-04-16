package storage

import (
	"LazyCatBot/internal/sirus"
	"sync"
)

type LeaderboardStorage struct {
	mu   sync.RWMutex
	// [RaidID][BossID][Spec]
	Data map[int]map[int]map[string][]sirus.LeaderboardPlayer 
}

func NewLeaderboardStorage() *LeaderboardStorage {
	return &LeaderboardStorage{
		Data: make(map[int]map[int]map[string][]sirus.LeaderboardPlayer),
	}
}

func (s *LeaderboardStorage) Update(raidID, bossID int, spec string, players []sirus.LeaderboardPlayer) {
    s.mu.Lock()
    defer s.mu.Unlock()

    if s.Data[raidID] == nil {
        s.Data[raidID] = make(map[int]map[string][]sirus.LeaderboardPlayer)
    }
    if s.Data[raidID][bossID] == nil {
        s.Data[raidID][bossID] = make(map[string][]sirus.LeaderboardPlayer)
    }

    s.Data[raidID][bossID][spec] = players
}