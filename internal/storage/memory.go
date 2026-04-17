package storage

import (
	"LazyCatBot/internal/sirus"
	"sync"
)

type LeaderboardStorage struct {
	mu sync.RWMutex
	// [mapId][difficulty][encounterID][spec]
	Data map[int]map[int]map[int]map[string][]sirus.LeaderboardPlayer
}

func NewLeaderboardStorage() *LeaderboardStorage {
	return &LeaderboardStorage{
		Data: make(map[int]map[int]map[int]map[string][]sirus.LeaderboardPlayer),
	}
}

func (s *LeaderboardStorage) Update(mapId, difficulty, encounterID int, spec string, players []sirus.LeaderboardPlayer) {
	s.mu.Lock()
	defer s.mu.Unlock()

	if s.Data[mapId] == nil {
		s.Data[mapId] = make(map[int]map[int]map[string][]sirus.LeaderboardPlayer)
	}
	if s.Data[mapId][difficulty] == nil {
		s.Data[mapId][difficulty] = make(map[int]map[string][]sirus.LeaderboardPlayer)
	}
	if s.Data[mapId][difficulty][encounterID] == nil {
		s.Data[mapId][difficulty][encounterID] = make(map[string][]sirus.LeaderboardPlayer)
	}

	s.Data[mapId][difficulty][encounterID][spec] = players
}
