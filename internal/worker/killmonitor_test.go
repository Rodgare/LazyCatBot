package worker

import (
	"LazyCatBot/internal/models"
	"LazyCatBot/internal/storage"
	"context"
	"database/sql"
	"log/slog"
	"reflect"
	"testing"
	"time"

	_ "modernc.org/sqlite"
)

type MockReporter struct {
	LastChannelID string
	SentReport    models.BossKillReport
}

func (m *MockReporter) SendKillReport(channelID string, report models.BossKillReport) {
	m.LastChannelID = channelID
	m.SentReport = report
}

type MockSirusAPI struct{}

func (m *MockSirusAPI) FetchBossFightDetails(fightID int) (*models.BossFight, error) {
	bf := &models.BossFight{
		Order:     1,
		Encounter: 2,
	}
	bf.Data.MapName = "Цитадель Ледяной Короны"
	bf.Data.Guild.ID = 777
	bf.Data.Guild.Name = "Тестовая Гильдия"
	bf.Data.FightLength = "5m 30s"
	bf.Data.Attempts = 1
	bf.Data.KilledAt = "2026-05-29 12:00:00"

	bf.Data.Players = []models.Player{
		{
			GUID:    1,
			Name:    "ТестовыйКот",
			ClassID: 1,
			Spec:    1,
			Ilvl:    277,
			Dps:     15000,
		},
	}
	return bf, nil
}

func (m *MockSirusAPI) FetchGuildLatestBossKills(guildID int) (*models.LatestBossKills, error) {
	return nil, nil
}
func (m *MockSirusAPI) FetchPlayerLastActions(playerID int) (*models.PlayerLastActions, error) {
	return nil, nil
}
func (m *MockSirusAPI) FetchGuildMembers(guildID int) (*[]models.GuildMembers, error) {
	return nil, nil
}
func (m *MockSirusAPI) FetchActualRaids() (models.ActualSirusRaids, error) {
	return nil, nil
}
func (m *MockSirusAPI) FetchLeaderboard(raidID, bossID, classID, specID int, role string) ([]models.LeaderboardPlayer, error) {
	return nil, nil
}
func (m *MockSirusAPI) FetchMetasirusLeaderboard(mapID, bossID, difficulty int) ([]models.MetasirusLeaderboardPlayer, error) {
	return nil, nil
}

func TestSortKills(t *testing.T) {
	w := &Worker{}
	input := map[int]map[string]bool{
		42:  {"chan1": true},
		10:  {"chan2": true},
		105: {"chan1": true},
	}
	expected := []int{10, 42, 105}
	result := w.sortKills(input)
	if !reflect.DeepEqual(result, expected) {
		t.Errorf("Expected %v, but got %v", expected, result)
	}
}

func TestStartProcessor_WithMock(t *testing.T) {
	db, err := sql.Open("sqlite", ":memory:")
	if err != nil {
		t.Fatalf("Failed to open in-memory db: %v", err)
	}
	defer db.Close()

	_, err = db.Exec(`
		CREATE TABLE processed_kills (kill_id INTEGER, channel_id TEXT, PRIMARY KEY (kill_id, channel_id));
		CREATE TABLE subscribe (guild_id INTEGER, channel_id TEXT, discord_id TEXT, is_send INTEGER DEFAULT 1, PRIMARY KEY (guild_id, channel_id, discord_id));
		CREATE TABLE leaderboard (raid_id INTEGER, boss_id INTEGER, class_id INTEGER, spec_id INTEGER, player_name TEXT, ilvl INTEGER, guild_id INTEGER, zodiac INTEGER, category INTEGER, t4 INTEGER, role TEXT, dps INTEGER, hps INTEGER, PRIMARY KEY (raid_id, boss_id, class_id, spec_id, player_name));
	`)
	if err != nil {
		t.Fatalf("Failed to create tables: %v", err)
	}

	logger := slog.Default()
	realSubStore := storage.NewSubscribeStorage(db)
	realLbStore := storage.NewLeaderboardStorage(db, logger)

	err = realSubStore.Subscribe(777, "discord-channel-xyz", "some-guild-id")
	if err != nil {
		t.Fatalf("Failed to setup subscribe: %v", err)
	}

	mockReporter := &MockReporter{}
	mockSirus := &MockSirusAPI{}

	w := &Worker{
		reporter:    mockReporter,
		subStore:    realSubStore,
		lbStore:     realLbStore,
		sirusClient: mockSirus,
		killQueue:   make(chan KillJob, 10),
		logger:      logger,
	}

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	go w.StartProcessor(ctx)

	job := KillJob{
		KillID:   123,
		Channels: map[string]bool{"discord-channel-xyz": true},
	}
	w.killQueue <- job

	time.Sleep(20 * time.Millisecond)

	if mockReporter.LastChannelID != "discord-channel-xyz" {
		t.Errorf("Expected report to be sent to 'discord-channel-xyz', but got '%s'", mockReporter.LastChannelID)
	}

	if mockReporter.SentReport.GuildName != "Тестовая Гильдия" {
		t.Errorf("Expected guild name 'Тестовая Гильдия', but got '%s'", mockReporter.SentReport.GuildName)
	}
}
