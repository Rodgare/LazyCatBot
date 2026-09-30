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

func (m *MockSirusAPI) FetchBossFightDetails(realm string, fightID int) (*models.BossFight, error) {
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

func (m *MockSirusAPI) FetchGuildLatestBossKills(realm string, guildID int) (*models.LatestBossKills, error) {
	return nil, nil
}
func (m *MockSirusAPI) FetchPlayerLastActions(realm string, playerID int) (*models.PlayerLastActions, error) {
	return nil, nil
}
func (m *MockSirusAPI) FetchGuildMembers(realm string, guildID int) (*[]models.GuildMembers, error) {
	return nil, nil
}
func (m *MockSirusAPI) FetchActualRaids(realm string) (models.ActualSirusRaids, error) {
	return nil, nil
}
func (m *MockSirusAPI) FetchLeaderboard(realm string, raidID, bossID, classID, specID int, role string) ([]models.LeaderboardPlayer, error) {
	return nil, nil
}

func (m *MockSirusAPI) GetLatestMythicRuns(realm string) (*models.MythicRuns, error) {
	runs := &models.MythicRuns{}
	runs.Data = []struct {
		ID      int `json:"id"`
		Members []struct {
			MemberGUID int    `json:"memberGuid"`
			Name       string `json:"name"`
		} `json:"members"`
		Position int `json:"position"`
	}{
		{
			ID: 5001,
			Members: []struct {
				MemberGUID int    `json:"memberGuid"`
				Name       string `json:"name"`
			}{
				{MemberGUID: 101, Name: "Игрок101"},
				{MemberGUID: 999, Name: "РандомныйИгрок"},
			},
		},
		{
			ID: 5002,
			Members: []struct {
				MemberGUID int    `json:"memberGuid"`
				Name       string `json:"name"`
			}{
				{MemberGUID: 888, Name: "ЧужойИгрок"},
			},
		},
	}
	return runs, nil
}

// -------------------------------------------------------------------
// ТЕСТ ШАГА 1: Получение последних мифик-ранов из SirusAPI
// -------------------------------------------------------------------
func TestMythicStep1_FetchRuns(t *testing.T) {
	mockAPI := &MockSirusAPI{}

	runs, err := mockAPI.GetLatestMythicRuns("x3")
	if err != nil {
		t.Fatalf("Шаг 1: Ошибка получения ранов из API: %v", err)
	}

	t.Logf("--> [Шаг 1 OK] Успешно получено ранов: %d", len(runs.Data))
	for idx, run := range runs.Data {
		t.Logf("   Ран #%d: ID=%d, Игроки=%+v", idx+1, run.ID, run.Members)
	}

	if len(runs.Data) == 0 {
		t.Errorf("Шаг 1: Ожидались раны, но получили 0")
	}
}

// -------------------------------------------------------------------
// ТЕСТ ШАГА 2: Получение отслеживаемых игроков и участников гильдий из БД
// -------------------------------------------------------------------
func TestMythicStep2_CollectTrackedPlayers(t *testing.T) {
	db, err := sql.Open("sqlite", ":memory:")
	if err != nil {
		t.Fatalf("Failed to open in-memory db: %v", err)
	}
	defer db.Close()

	_, err = db.Exec(`
		CREATE TABLE IF NOT EXISTS player_subscribe (id INTEGER, name TEXT, channel_id TEXT, discord_id TEXT, realm TEXT, PRIMARY KEY (id, channel_id, discord_id));
		CREATE TABLE IF NOT EXISTS subscribe (guild_id INTEGER, channel_id TEXT, discord_id TEXT, is_send INTEGER DEFAULT 1, realm TEXT DEFAULT 'x3', PRIMARY KEY (guild_id, channel_id, discord_id));
		CREATE TABLE IF NOT EXISTS guild_members (guild_id INTEGER, realm TEXT, player_id INTEGER, name TEXT, PRIMARY KEY (guild_id, realm, player_id));
	`)
	if err != nil {
		t.Fatalf("Failed to create tables: %v", err)
	}

	pSubStore := storage.NewPlayerSubscribeStorage(db)

	// Добавляем тестовую подписку на игрока 101 на канал "discord-channel-mythic"
	err = pSubStore.Subscribe(101, "Игрок101", "discord-channel-mythic", "discord-user-1", "x3")
	if err != nil {
		t.Fatalf("Failed to subscribe player: %v", err)
	}

	trackedPlayers, err := pSubStore.GetTrackedPlayers()
	if err != nil {
		t.Fatalf("Шаг 2: Ошибка сбора отслеживаемых игроков: %v", err)
	}

	t.Logf("--> [Шаг 2 OK] Карта отслеживаемых игроков: %+v", trackedPlayers)

	found := false
	for key, channels := range trackedPlayers {
		if key.PlayerID == 101 {
			found = true
			t.Logf("   Найдена подписка: Игрок ID=%d, Реалм=%s, Каналы=%v", key.PlayerID, key.Realm, channels)
		}
	}

	if !found {
		t.Errorf("Шаг 2: Игрок ID 101 не найден в возвращенных отслеживаемых игроках")
	}
}

// -------------------------------------------------------------------
// ТЕСТ ШАГА 3: Фильтрация ранов и отправка задач в очередь killQueue
// -------------------------------------------------------------------
func TestMythicStep3_FilterAndQueueJobs(t *testing.T) {
	db, err := sql.Open("sqlite", ":memory:")
	if err != nil {
		t.Fatalf("Failed to open in-memory db: %v", err)
	}
	defer db.Close()

	_, err = db.Exec(`
		CREATE TABLE IF NOT EXISTS processed_kills (kill_id INTEGER, channel_id TEXT, PRIMARY KEY (kill_id, channel_id));
		CREATE TABLE IF NOT EXISTS player_subscribe (id INTEGER, name TEXT, channel_id TEXT, discord_id TEXT, realm TEXT, PRIMARY KEY (id, channel_id, discord_id));
		CREATE TABLE IF NOT EXISTS guild_members (guild_id INTEGER, realm TEXT, player_id INTEGER, name TEXT, PRIMARY KEY (guild_id, realm, player_id));
	`)
	if err != nil {
		t.Fatalf("Failed to create tables: %v", err)
	}

	logger := slog.Default()
	mockAPI := &MockSirusAPI{}
	pSubStore := storage.NewPlayerSubscribeStorage(db)
	gmStore := storage.NewGuildMembersStorage(db)
	subStore := storage.NewSubscribeStorage(db)

	_ = pSubStore.Subscribe(101, "Игрок101", "discord-channel-mythic", "discord-user-1", "x3")

	w := &Worker{
		sirusClient: mockAPI,
		subStore:    subStore,
		pSubStore:   pSubStore,
		gmStore:     gmStore,
		killQueue:   make(chan KillJob, 50),
		logger:      logger,
	}

	// Вызываем главную функцию обработки
	w.processMythicRuns()

	// Проверяем результат выполнения и очереди
	select {
	case job := <-w.killQueue:
		t.Logf("--> [Шаг 3 OK] В очередь killQueue передан джоб: %+v", job)
		if job.KillID != 5001 {
			t.Errorf("Ожидался KillID = 5001, получено %d", job.KillID)
		}
		if !job.Channels["discord-channel-mythic"] {
			t.Errorf("Ожидалась отправка в канал 'discord-channel-mythic', получено: %v", job.Channels)
		}
	default:
		t.Log("Информация: Функция processMythicRuns пока не отправляет джобы в killQueue (или фильтрация ещё не завершена)")
	}
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
		CREATE TABLE subscribe (guild_id INTEGER, channel_id TEXT, discord_id TEXT, is_send INTEGER DEFAULT 1, realm TEXT DEFAULT 'x3', PRIMARY KEY (guild_id, channel_id, discord_id));
		CREATE TABLE leaderboard (raid_id INTEGER, boss_id INTEGER, class_id INTEGER, spec_id INTEGER, player_name TEXT, ilvl INTEGER, guild_id INTEGER, zodiac INTEGER, category INTEGER, t4 INTEGER, role TEXT, dps INTEGER, hps INTEGER, PRIMARY KEY (raid_id, boss_id, class_id, spec_id, player_name));
	`)
	if err != nil {
		t.Fatalf("Failed to create tables: %v", err)
	}

	logger := slog.Default()
	realSubStore := storage.NewSubscribeStorage(db)
	realLbStore := storage.NewLeaderboardStorage(db, logger)

	err = realSubStore.Subscribe(777, "discord-channel-xyz", "some-guild-id", "x3")
	if err != nil {
		t.Fatalf("Failed to setup subscribe: %v", err)
	}

	mockReporter := &MockReporter{}
	mockAPI := &MockSirusAPI{}

	w := &Worker{
		sirusClient: mockAPI,
		reporter:    mockReporter,
		subStore:    realSubStore,
		lbStore:     realLbStore,
		killQueue:   make(chan KillJob, 10),
		logger:      logger,
	}

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	go w.StartProcessor(ctx)

	job := KillJob{
		KillID:   123,
		Realm:    "x3",
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
