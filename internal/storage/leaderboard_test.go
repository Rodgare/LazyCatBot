package storage

import (
	"LazyCatBot/internal/sirus"
	"database/sql"
	"testing"

	_ "modernc.org/sqlite" // или твой драйвер sqlite
)

func TestGetRank(t *testing.T) {
	db, err := sql.Open("sqlite", ":memory:")
	if err != nil {
		t.Fatalf("Не удалось открыть БД: %v", err)
	}
	defer db.Close()

	s := NewLeaderboardStorage(db)
	if err := s.InitDB(); err != nil {
		t.Fatalf("Не удалось инициализировать БД: %v", err)
	}

	testPlayers := []sirus.LeaderboardPlayer{
		{Name: "Топ_1_Спека", ClassID: 1, SpecID: 0, Ilvl: 242, Dps: 50000},
		{Name: "Топ_2_Спека", ClassID: 1, SpecID: 0, Ilvl: 241, Dps: 40000},
		{Name: "Другой_Спек", ClassID: 1, SpecID: 1, Ilvl: 240, Dps: 30000},
		{Name: "Другой_Брекет", ClassID: 1, SpecID: 0, Ilvl: 238, Dps: 20000},
		{Name: "Dummy5", ClassID: 1, SpecID: 0, Ilvl: 200, Dps: 1000},
		{Name: "Dummy6", ClassID: 1, SpecID: 0, Ilvl: 200, Dps: 1000},
		{Name: "Dummy7", ClassID: 1, SpecID: 0, Ilvl: 200, Dps: 1000},
		{Name: "Dummy8", ClassID: 1, SpecID: 0, Ilvl: 200, Dps: 1000},
		{Name: "Dummy9", ClassID: 1, SpecID: 0, Ilvl: 200, Dps: 1000},
		{Name: "Dummy10", ClassID: 1, SpecID: 0, Ilvl: 200, Dps: 1000},
	}
	if err := s.UpdateLeaderboardStorage(11, 1, testPlayers); err != nil {
		t.Fatalf("Ошибка при наполнении БД: %v", err)
	}

	p := sirus.Player{
		ClassID: 1,
		Spec:    0,
		Ilvl:    241,
		Dps:     40000,
	}

	specRank, specPrcnt, classRank, classPrcnt, ilvlRank, ilvlPrcnt, overallRank, overallPrcnt, err := s.GetRank(11, 1, p)
	if err != nil {
		t.Fatalf("Ошибка выполнения GetRank: %v", err)
	}

	if specRank != 2 {
		t.Errorf("Ожидали SpecRank 2, получили %d", specRank)
	}

	if ilvlRank != 2 {
		t.Errorf("Ожидали IlvlRank 2, получили %d", ilvlRank)
	}

	if classRank != 2 {
		t.Errorf("Ожидали ClassRank 2, получили %d", classRank)
	}

	if specPrcnt != 88 {
		t.Errorf("Ожидали SpecPercentile 88, получили %d", specPrcnt)
	}

	if classPrcnt != 90 {
		t.Errorf("Ожидали ClassPercentile 90, получили %d", classPrcnt)
	}

	if overallRank != 2 {
		t.Errorf("Ожидали OverallRank 2, получили %d", overallRank)
	}

	if overallPrcnt != 90 {
		t.Errorf("Ожидали OverallPercentile 90, получили %d", overallPrcnt)
	}

	t.Logf("Тест пройден! Ranks: Spec=%d, Class=%d, Ilvl=%d, Overall=%d. Percentiles: Spec=%d%%, Class=%d%%, Ilvl=%d%%, Overall=%d%%",
		specRank, classRank, ilvlRank, overallRank, specPrcnt, classPrcnt, ilvlPrcnt, overallPrcnt)
}
