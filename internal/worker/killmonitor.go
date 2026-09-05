package worker

import (
	"LazyCatBot/internal/models"
	"LazyCatBot/internal/sirus"
	"LazyCatBot/internal/storage"
	"context"
	"encoding/json"
	"log/slog"
	"os"
	"sort"
	"time"
)

type Reporter interface {
	SendKillReport(channelID string, report models.BossKillReport)
}

type SubscribeStore interface {
	IsKillProcessed(id int, ch string) bool
	MarkKillProcessed(id int, ch string) error
	IsReportsEnabled(guild int, channelID string) bool
	GetTrackedGuilds() (map[storage.TrackedGuildKey][]string, error)
}

type SirusAPI interface {
	FetchBossFightDetails(realm string, fightID int) (*models.BossFight, error)
	FetchGuildLatestBossKills(realm string, guildID int) (*models.LatestBossKills, error)
	FetchPlayerLastActions(realm string, playerID int) (*models.PlayerLastActions, error)
	FetchGuildMembers(realm string, guildID int) (*[]models.GuildMembers, error)
	FetchActualRaids(realm string) (models.ActualSirusRaids, error)
	FetchLeaderboard(realm string, raidID, bossID, classID, specID int, role string) ([]models.LeaderboardPlayer, error)
	FetchMetasirusLeaderboard(realm string, mapID, bossID, difficulty int) ([]models.MetasirusLeaderboardPlayer, error)
}

type KillJob struct {
	KillID   int
	Realm    string
	Channels map[string]bool
}

type Worker struct {
	sirusClient SirusAPI
	lbStore     *storage.LeaderboardStorage
	subStore    SubscribeStore
	pSubStore   *storage.PlayerSubscribeStorage
	gmStore     *storage.GuildMembersStorage
	arStore     *storage.ActualRaidsStorage
	reporter    Reporter
	killQueue   chan KillJob
	logger      *slog.Logger
}

func NewWorker(
	sc SirusAPI,
	lb *storage.LeaderboardStorage,
	sub SubscribeStore,
	gm *storage.GuildMembersStorage,
	ps *storage.PlayerSubscribeStorage,
	ar *storage.ActualRaidsStorage,
	reporter Reporter,
	logger *slog.Logger,
) *Worker {
	return &Worker{
		sirusClient: sc,
		lbStore:     lb,
		subStore:    sub,
		pSubStore:   ps,
		gmStore:     gm,
		arStore:     ar,
		reporter:    reporter,
		killQueue:   make(chan KillJob, 100),
		logger:      logger,
	}
}

func (w *Worker) GuildKillMonitor(ctx context.Context) {
	for {
		select {
		case <-ctx.Done():
			w.logger.Info("[KillMonitor] Guild monitor stopped")
			return
		default:
		}

		guilds, err := w.subStore.GetTrackedGuilds()
		if err != nil {
			w.logger.Error("[KillMonitor] Getting tracked guilds err", "error", err)
			select {
			case <-ctx.Done():
				return
			case <-time.After(1 * time.Minute):
			}
			continue
		}

		kills, killRealms := w.getGuildKills(guilds)
		sortedKillsIDs := w.sortKills(kills)

		for _, killID := range sortedKillsIDs {
			select {
			case <-ctx.Done():
				return
			case w.killQueue <- KillJob{KillID: killID, Realm: killRealms[killID], Channels: kills[killID]}:
			}
		}

		select {
		case <-ctx.Done():
			w.logger.Info("[KillMonitor] Guild monitor stopped")
			return
		case <-time.After(1 * time.Minute):
		}
	}
}

func (w *Worker) PlayerKillMonitor(ctx context.Context) {
	for {
		select {
		case <-ctx.Done():
			w.logger.Info("[KillMonitor] Player monitor stopped")
			return
		default:
		}

		players, err := w.pSubStore.GetTrackedPlayers()
		if err != nil {
			w.logger.Error("[KillMonitor] Getting tracked players err", "error", err)
			select {
			case <-ctx.Done():
				return
			case <-time.After(1 * time.Minute):
			}
			continue
		}

		kills, killRealms := w.getPlayerKills(players)
		sortedKillsIDs := w.sortKills(kills)

		for _, killID := range sortedKillsIDs {
			select {
			case <-ctx.Done():
				return
			case w.killQueue <- KillJob{KillID: killID, Realm: killRealms[killID], Channels: kills[killID]}:
			}
		}

		select {
		case <-ctx.Done():
			w.logger.Info("[KillMonitor] Player monitor stopped")
			return
		case <-time.After(1 * time.Minute):
		}
	}
}

func (w *Worker) StartProcessor(ctx context.Context) {
	for job := range w.killQueue {
		select {
		case <-ctx.Done():
			w.logger.Info("[KillMonitor] Processor stopped")
			return
		default:
		}

		var report *models.BossKillReport

		for ch := range job.Channels {
			if w.subStore.IsKillProcessed(job.KillID, ch) {
				continue
			}

			if report == nil {
				enrichedKill, err := w.sirusClient.FetchBossFightDetails(job.Realm, job.KillID)
				if err != nil {
					w.logger.Error("[Processor] Fetch Boss Fight Details err", "error", err)
					break
				}

				rep := w.createReport(enrichedKill, job.KillID, job.Realm)
				report = &rep
			}

			if w.subStore.IsReportsEnabled(report.GuildID, ch) {
				w.reporter.SendKillReport(ch, *report)
			}
			w.subStore.MarkKillProcessed(job.KillID, ch)

			select {
			case <-ctx.Done():
				w.logger.Info("[KillMonitor] Processor stopped")
				return
			case <-time.After(300 * time.Millisecond):
			}
		}
	}
}

func (w *Worker) getGuildKills(data map[storage.TrackedGuildKey][]string) (map[int]map[string]bool, map[int]string) {
	kills := make(map[int]map[string]bool)
	killRealms := make(map[int]string)

	for key, channels := range data {
		gKills, err := w.sirusClient.FetchGuildLatestBossKills(key.Realm, key.GuildID)
		if err != nil {
			w.logger.Error("[KillMonitor] Fetch Guild Latest BossKills err", "error", err, "guild_id", key.GuildID, "realm", key.Realm)
			continue
		}

		for _, kill := range gKills.Data {
			for _, ch := range channels {
				id := kill.KillID
				if !w.subStore.IsKillProcessed(id, ch) {
					if _, ok := kills[id]; !ok {
						kills[id] = make(map[string]bool)
					}

					kills[id][ch] = true
					killRealms[id] = key.Realm
				}
			}
		}
		time.Sleep(300 * time.Millisecond)
	}

	return kills, killRealms
}

func (w *Worker) getPlayerKills(data map[storage.TrackedPlayerKey][]string) (map[int]map[string]bool, map[int]string) {
	kills := make(map[int]map[string]bool)
	killRealms := make(map[int]string)

	for key, channels := range data {
		pKills, err := w.sirusClient.FetchPlayerLastActions(key.Realm, key.PlayerID)
		if err != nil || pKills == nil {
			w.logger.Error("[KillMonitor] pKills is nil or Fetch Player Latest BossKills err", "error", err, "player_id", key.PlayerID, "realm", key.Realm)
			continue
		}
		lastKills := *pKills

		if len(lastKills) >= 10 {
			lastKills = lastKills[:10]
		}

		for _, kill := range lastKills {
			if kill.Type != "bosskill" || !sirus.IsPlayerKillToday(kill.Date) {
				continue
			}
			for _, ch := range channels {
				id := kill.FightID

				if !w.subStore.IsKillProcessed(id, ch) {
					if _, ok := kills[id]; !ok {
						kills[id] = make(map[string]bool)
					}

					kills[id][ch] = true
					killRealms[id] = key.Realm
				}
			}
		}
		time.Sleep(300 * time.Millisecond)
	}

	return kills, killRealms
}

func (w *Worker) createReport(fight *models.BossFight, killID int, realm string) models.BossKillReport {
	if realm == "" {
		realm = "x3"
	}
	totalDps := 0
	totalHps := 0
	for _, p := range fight.Data.Players {
		totalDps += p.Dps
		totalHps += p.Hps
	}

	report := models.BossKillReport{
		MapName:   fight.Data.MapName,
		BossName:  sirus.GetBossName(fight.Order, fight.Encounter),
		RaidOrder: fight.Order,
		KillID:    killID,
		Duration:  fight.Data.FightLength,
		Attempts:  fight.Data.Attempts,
		KilledAt:  fight.Data.KilledAt,
		GuildID:   fight.Data.Guild.ID,
		GuildName: fight.Data.Guild.Name,
		TotalDps:  totalDps,
		TotalHps:  totalHps,
		Realm:     realm,
	}

	for _, loot := range fight.Data.Loots {
		lootReport := models.LootReport{
			ID:    loot.Entry,
			Name:  loot.Item.Name,
			Count: loot.Count,
			Icon:  loot.Item.Icon,
		}
		report.Loots = append(report.Loots, lootReport)
	}

	for _, p := range fight.Data.Players {
		specName := sirus.GetSpecName(p.ClassID, p.Spec)
		t4Count := sirus.GetT4Count(p.Itemset)
		role := sirus.GetRoleString(p.ClassID, p.Spec)

		err := w.lbStore.UpsertPlayer(realm, fight.Order, fight.Encounter, p)
		if err != nil {
			w.logger.Error("Upsert player in db error", "error", err)
		}

		playerReport := models.PlayerReport{
			Name:     p.Name,
			Dps:      p.Dps,
			Hps:      p.Hps,
			Ilvl:     p.Ilvl,
			SpecID:   p.Spec,
			ClassID:  p.ClassID,
			SpecName: specName,
			T4:       t4Count,
			Role:     sirus.GetRole(p.ClassID, p.Spec),
			Zodiac:   p.Zodiac.ID,
			Category: p.Category,
		}

		pRole := "dps"
		if role == "hps" {
			pRole = "hps"
		}

		playerReport, err = w.lbStore.GetPlayerRank(fight.Order, fight.Encounter, playerReport, pRole)
		if err != nil {
			w.logger.Error("Getting player rank error", "error", err)
		}
		report.Players = append(report.Players, playerReport)
	}

	return report
}

func (w *Worker) makeMockReport() models.BossKillReport {
	var report models.BossKillReport

	data, err := os.ReadFile("internal/sirus/testdata/mock_sirus_boss_fight.json")
	if err != nil {
		w.logger.Error("Mock json read error", "error", err)
	}
	json.Unmarshal(data, &report)

	return report
}

func (w *Worker) sortKills(kills map[int]map[string]bool) []int {
	ids := make([]int, 0, len(kills))

	for id := range kills {
		ids = append(ids, id)
	}

	sort.Ints(ids)

	return ids
}
