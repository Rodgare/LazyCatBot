package storage

import (
	"LazyCatBot/internal/sirus"
	"sync"
)

type LeaderboardStorage struct {
	mu sync.RWMutex
	// [raidOrder][encounter][class][spec]
	Data map[int]map[int]map[int]map[int][]sirus.LeaderboardPlayer
}

func NewLeaderboardStorage() *LeaderboardStorage {
	return &LeaderboardStorage{
		Data: make(map[int]map[int]map[int]map[int][]sirus.LeaderboardPlayer),
	}
}

func (s *LeaderboardStorage) UpdateLeaderboardStorage(raidOrder, encounter, class, spec int, players []sirus.LeaderboardPlayer) {
	s.mu.Lock()
	defer s.mu.Unlock()

	if s.Data[raidOrder] == nil {
		s.Data[raidOrder] = make(map[int]map[int]map[int][]sirus.LeaderboardPlayer)
	}
	if s.Data[raidOrder][encounter] == nil {
		s.Data[raidOrder][encounter] = make(map[int]map[int][]sirus.LeaderboardPlayer)
	}
	if s.Data[raidOrder][encounter][class] == nil {
		s.Data[raidOrder][encounter][class] = make(map[int][]sirus.LeaderboardPlayer)
	}

	s.Data[raidOrder][encounter][class][spec] = players
}
