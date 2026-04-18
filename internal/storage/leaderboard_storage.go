package storage

import (
	"LazyCatBot/internal/sirus"
	"fmt"
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

func (s *LeaderboardStorage) GetSpecRank(raid, boss, class, spec, dps int) int {
	s.mu.RLock()
	defer s.mu.RUnlock()
	if _, ok := s.Data[raid]; !ok {
		fmt.Printf("No data in LeaderboardStorage for raid id: %v\n", raid)
		return 0
	}
	if _, ok := s.Data[raid][boss]; !ok {
		fmt.Printf("No data in LeaderboardStorage for boss id: %v\n", boss)
		return 0
	}
	if _, ok := s.Data[raid][boss][class]; !ok {
		fmt.Printf("No data in LeaderboardStorage for class id: %v\n", class)
		return 0
	}
	if _, ok := s.Data[raid][boss][class][spec]; !ok {
		fmt.Printf("No data in LeaderboardStorage for spec id: %v\n", spec)
		return 0
	}
	leaderboard := s.Data[raid][boss][class][spec]
	for _, player := range leaderboard {
		if dps > player.Dps {
			return player.Rank
		}
	}

	return len(leaderboard) + 1
}
