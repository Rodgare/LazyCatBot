package worker

import (
	"LazyCatBot/internal/config"
	"LazyCatBot/internal/models"
	"LazyCatBot/internal/sirus"
	"LazyCatBot/internal/storage"
	"context"
	"encoding/json"
	"log/slog"
	"os"
	"sort"
	"strings"
	"sync"
	"time"
)

type Reporter interface {
	SendKillReport(channelID string, report models.BossKillReport) error
	SendMythicReport(channelID string, report *models.MythicReport) error
	SendMythicTopMock(channelID string) error
}

type SubscribeStore interface {
	IsKillProcessed(id int, ch string) bool
	MarkKillProcessed(id int, ch string) error
	CleanupProcessedKills(olderThan time.Duration) error
	DisableReportsForChannel(ch string) error
	SetGuildName(guildID int, realm, name string) error
	GetGuildName(guildID int, realm string) (string, error)
	IsReportsEnabled(guild int, channelID string) bool
	GetTrackedGuilds() (map[storage.TrackedGuildKey][]string, error)
	IsMythicReportsEnabled(ch string) bool
}

func isMissingAccess(err error) bool {
	if err == nil {
		return false
	}
	msg := err.Error()
	return strings.Contains(msg, "Missing Access") || strings.Contains(msg, "50001")
}

type SirusAPI interface {
	FetchBossFightDetails(realm string, fightID int) (*models.BossFight, error)
	FetchGuildLatestBossKills(realm string, guildID int) (*models.LatestBossKills, error)
	FetchPlayerLastActions(realm string, playerID int) (*models.PlayerLastActions, error)
	FetchGuildMembers(realm string, guildID int) (*[]models.GuildMembers, string, error)
	FetchCharacter(realm string, id int) (*models.CharacterData, error)
	FetchActualRaids(realm string) (models.ActualSirusRaids, error)
	FetchLeaderboard(realm string, raidID, bossID, classID, specID int, role string) ([]models.LeaderboardPlayer, error)
	GetLatestMythicRuns(realm string) (*models.MythicRuns, error)
	FetchMythicRunDetails(realm string, runID int) (*models.MythicRun, error)
	FetchChallengeScores(realm string, season, weekID int) ([]models.MythicScorePlayer, error)
}

type KillJob struct {
	KillID       int
	Realm        string
	Channels     map[string]bool
	Type         string
	MythicReport *models.MythicReport
}

type Worker struct {
	sirusClient  SirusAPI
	lbStore      *storage.LeaderboardStorage
	mythLbStore  *storage.MythicLeaderboardStorage
	subStore     SubscribeStore
	pSubStore    *storage.PlayerSubscribeStorage
	gmStore      *storage.GuildMembersStorage
	charStore    *storage.CharacterCacheStorage
	arStore      *storage.ActualRaidsStorage
	reporter     Reporter
	killQueue    chan KillJob
	charPriority chan PriorityChar
	apiLimiter   *RateLimiter
	logger       *slog.Logger
	cfg          *config.Config
}

func NewWorker(
	sc SirusAPI,
	lb *storage.LeaderboardStorage,
	sub SubscribeStore,
	gm *storage.GuildMembersStorage,
	ps *storage.PlayerSubscribeStorage,
	cc *storage.CharacterCacheStorage,
	ar *storage.ActualRaidsStorage,
	reporter Reporter,
	logger *slog.Logger,
	cfg *config.Config,
) *Worker {
	return &Worker{
		sirusClient:  sc,
		lbStore:      lb,
		subStore:     sub,
		pSubStore:    ps,
		gmStore:      gm,
		charStore:    cc,
		arStore:      ar,
		reporter:     reporter,
		killQueue:    make(chan KillJob, 100),
		charPriority: make(chan PriorityChar, 1000),
		apiLimiter:   NewRateLimiter(2, 1),
		logger:       logger,
		cfg:          cfg,
	}
}

// SetMythicLeaderboardStore wires the optional mythic leaderboard store used by
// the challenge scores sync and report enrichment.
func (w *Worker) SetMythicLeaderboardStore(ml *storage.MythicLeaderboardStorage) {
	w.mythLbStore = ml
}

func (w *Worker) GuildKillMonitor(ctx context.Context) {
	if w.cfg.IsDebug {
		w.logger.Info("[DEBUG] Sending mock raid report...")
		mockReport := w.makeMockReport()
		if err := w.reporter.SendKillReport(w.cfg.DebugChannelID, mockReport); err != nil {
			w.logger.Error("[DEBUG] Send mock raid report err", "error", err)
		}
	}
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
			case w.killQueue <- KillJob{
				KillID:   killID,
				Realm:    killRealms[killID],
				Channels: kills[killID],
				Type:     "raid",
			}:
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
			case w.killQueue <- KillJob{
				KillID:   killID,
				Realm:    killRealms[killID],
				Channels: kills[killID],
				Type:     "raid",
			}:
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

func (w *Worker) MythicRunsMonitor(ctx context.Context) {
	monitorStartTime := int(time.Now().Unix())

	if w.cfg.IsDebug {
		w.logger.Info("[DEBUG] Pushing mock mythic run to killQueue...")
		mockReport := w.makeMockMythicReport()
		if err := w.reporter.SendMythicReport(w.cfg.DebugChannelID, mockReport); err != nil {
			w.logger.Error("[DEBUG] Send mock mythic report err", "error", err)
		}
		if err := w.reporter.SendMythicTopMock(w.cfg.DebugChannelID); err != nil {
			w.logger.Error("[DEBUG] Send mock mythic top err", "error", err)
		}
	}
	for {
		select {
		case <-ctx.Done():
			w.logger.Info("[KillMonitor] Mythic Runs Monitor stopped")
			return
		default:
		}

		allTrackedPlayers := make(map[storage.TrackedPlayerKey]map[string]bool)

		trackedPlayers, err := w.pSubStore.GetTrackedPlayers()
		if err != nil {
			w.logger.Error("[KillMonitor] GetTrackedPlayers err", "error", err)
			continue
		}

		for key, channels := range trackedPlayers {
			if _, exists := allTrackedPlayers[key]; !exists {
				allTrackedPlayers[key] = make(map[string]bool)
			}
			for _, channelID := range channels {
				allTrackedPlayers[key][channelID] = true
			}
		}

		guilds, err := w.subStore.GetTrackedGuilds()
		if err != nil {
			w.logger.Error("[KillMonitor] GetTrackedGuilds err", "error", err)
			continue
		}

		for key, channels := range guilds {
			guildPlayers, err := w.gmStore.GetPlayersByGuildID(key.Realm, key.GuildID)
			if err != nil {
				w.logger.Error("[KillMonitor] GetPlayersByGuildID err", "error", err)
				continue
			}
			for _, playerID := range guildPlayers {
				playerKey := storage.TrackedPlayerKey{
					PlayerID: playerID,
					Realm:    key.Realm,
				}

				if _, exists := allTrackedPlayers[playerKey]; !exists {
					allTrackedPlayers[playerKey] = make(map[string]bool)
				}
				for _, channel := range channels {
					allTrackedPlayers[playerKey][channel] = true
				}
			}
		}

		realm := w.cfg.DefaultRealm
		w.apiLimiter.Wait()
		latestRuns, err := w.sirusClient.GetLatestMythicRuns(realm)
		if err != nil {
			w.logger.Error("[KillMonitor] GetLatestMythicRuns err", "error", err)
			continue
		}

		//[runID][channelID] = MythicReport
		matchedRuns := make(map[int]map[string]models.MythicReport)

		for _, run := range latestRuns.Data {
			if run.DateTime < monitorStartTime {
				continue
			}

			var (
				report models.MythicReport
				mapped bool
			)
			for _, player := range run.Members {
				playerKey := storage.TrackedPlayerKey{
					PlayerID: player.MemberGUID,
					Realm:    realm,
				}
				channels, ok := allTrackedPlayers[playerKey]
				if !ok {
					continue
				}
				if !mapped {
					report = run.ToReport()
					report.Realm = realm
					mapped = true
				}
				if matchedRuns[run.ID] == nil {
					matchedRuns[run.ID] = make(map[string]models.MythicReport)
				}
				for channel := range channels {
					matchedRuns[run.ID][channel] = report
				}
			}
		}

		for runID, channelReports := range matchedRuns {
			channels := make(map[string]bool, len(channelReports))
			var report models.MythicReport

			for chID, rep := range channelReports {
				channels[chID] = true
				report = rep
			}

			select {
			case <-ctx.Done():
				return
			case w.killQueue <- KillJob{
				KillID:       runID,
				Realm:        realm,
				Channels:     channels,
				MythicReport: &report,
				Type:         "myth",
			}:
			}
		}

		select {
		case <-ctx.Done():
			w.logger.Info("[KillMonitor] Mythic Runs Monitor stopped")
			return
		case <-time.After(1 * time.Minute):
		}
	}
}

const killQueueOverloadThreshold = 10

func (w *Worker) StartProcessor(ctx context.Context) {
	queueOverloaded := false
	for {
		select {
		case <-ctx.Done():
			w.logger.Info("[KillMonitor] Processor stopped")
			return
		case job, ok := <-w.killQueue:
			if !ok {
				w.logger.Info("[KillMonitor] Kill queue closed, processor stopped")
				return
			}

			queued := len(w.killQueue)
			if queued >= killQueueOverloadThreshold {
				if !queueOverloaded {
					queueOverloaded = true
					w.logger.Warn("[KillMonitor] Kill queue is accumulating",
						"queued", queued, "threshold", killQueueOverloadThreshold,
					)
				}
			} else {
				if queueOverloaded {
					queueOverloaded = false
					w.logger.Info("[KillMonitor] Kill queue drained", "queued", queued)
				}
			}

			switch job.Type {
			case "myth":
				w.processMythicRuns(job)
			default:
				w.processRaids(job)
			}

			select {
			case <-ctx.Done():
				w.logger.Info("[KillMonitor] Processor stopped")
				return
			case <-time.After(300 * time.Millisecond):
			}
		}
	}
}

func (w *Worker) processRaids(job KillJob) {
	var report *models.BossKillReport

	for ch := range job.Channels {
		if w.subStore.IsKillProcessed(job.KillID, ch) {
			continue
		}

		if report == nil {
			w.apiLimiter.Wait()
			enrichedKill, err := w.sirusClient.FetchBossFightDetails(job.Realm, job.KillID)
			if err != nil {
				w.logger.Error("[Processor] Fetch Boss Fight Details err", "error", err)
				break
			}

			rep := w.createReport(enrichedKill, job.KillID, job.Realm)
			w.enrichRaidCharacters(&rep, job.Realm)
			report = &rep

			for _, p := range enrichedKill.Data.Players {
				w.enqueueCharRefreshIfStale(job.Realm, p.Name, p.GUID)
			}
		}

		if w.subStore.IsReportsEnabled(report.GuildID, ch) {
			if err := w.reporter.SendKillReport(ch, *report); err != nil {
				if isMissingAccess(err) {
					w.subStore.DisableReportsForChannel(ch)
					w.logger.Warn("[Processor] Reports disabled for inaccessible channel", "error", err, "kill_id", job.KillID, "channel", ch)
				} else {
					w.logger.Error("[Processor] Send kill report err", "error", err, "kill_id", job.KillID, "channel", ch)
				}
				continue
			}
		}
		w.subStore.MarkKillProcessed(job.KillID, ch)

	}
}

func (w *Worker) processMythicRuns(job KillJob) {
	if job.MythicReport.HasRunLog {
		w.apiLimiter.Wait()
		runDetails, err := w.sirusClient.FetchMythicRunDetails(job.Realm, job.KillID)
		if err != nil {
			w.logger.Error("[Processor] Fetch Mythic Run Details err", "error", err)
			return
		}
		job.MythicReport.Enrich(runDetails)

		if job.MythicReport.HasCombatDetails() {
			job.MythicReport.ApplyCombat(runDetails)
		}
	}
	w.enrichMythicCharacters(job)
	for _, m := range job.MythicReport.Members {
		w.enqueueCharRefreshIfStale(job.Realm, m.Name, m.MemberGUID)
	}
	for ch := range job.Channels {
		if w.subStore.IsKillProcessed(job.KillID, ch) {
			continue
		}

		if w.subStore.IsMythicReportsEnabled(ch) {
			if err := w.reporter.SendMythicReport(ch, job.MythicReport); err != nil {
				if isMissingAccess(err) {
					w.subStore.DisableReportsForChannel(ch)
					w.logger.Warn("[Processor] Mythic reports disabled for inaccessible channel", "error", err, "run_id", job.KillID, "channel", ch)
				} else {
					w.logger.Error("[Processor] Send mythic report err", "error", err, "run_id", job.KillID, "channel", ch)
				}
				continue
			}
		}
		w.subStore.MarkKillProcessed(job.KillID, ch)
	}

}

func (w *Worker) enrichMythicCharacters(job KillJob) {
	if job.MythicReport == nil {
		return
	}
	for i := range job.MythicReport.Members {
		m := &job.MythicReport.Members[i]
		c, err := w.charStore.Get(job.Realm, m.Name)
		if err != nil {
			continue
		}
		m.MythicRating = c.MythicRating
		m.BlackDiamonds = c.BlackDiamonds
		m.Title = c.Title

		if w.mythLbStore != nil {
			if _, _, percentile, err := w.mythLbStore.GetPlayerMythicRank(
				job.Realm, w.cfg.MythicSeason, w.cfg.MythicWeekID,
				m.ClassID, m.SpecID, m.MythicRating,
			); err == nil {
				m.MythicRatingPercentile = percentile
			}
		}
	}
}

func (w *Worker) enrichRaidCharacters(report *models.BossKillReport, realm string) {
	if report == nil {
		return
	}
	for i := range report.Players {
		p := &report.Players[i]
		c, err := w.charStore.Get(realm, p.Name)
		if err != nil {
			continue
		}
		p.MythicRating = c.MythicRating
		p.BlackDiamonds = c.BlackDiamonds
		p.Title = c.Title
	}
}

func (w *Worker) getGuildKills(data map[storage.TrackedGuildKey][]string) (map[int]map[string]bool, map[int]string) {
	kills := make(map[int]map[string]bool)
	killRealms := make(map[int]string)

	var mu sync.Mutex
	keys := make([]storage.TrackedGuildKey, 0, len(data))
	for key := range data {
		keys = append(keys, key)
	}

	const workers = 6
	sem := make(chan struct{}, workers)
	var wg sync.WaitGroup

	for _, key := range keys {
		wg.Add(1)
		sem <- struct{}{}
		go func(key storage.TrackedGuildKey) {
			defer wg.Done()
			defer func() { <-sem }()

			channels := data[key]
			w.apiLimiter.Wait()
			gKills, err := w.sirusClient.FetchGuildLatestBossKills(key.Realm, key.GuildID)
			if err != nil {
				w.logger.Error("[KillMonitor] Fetch Guild Latest BossKills err", "error", err, "guild_id", key.GuildID, "realm", key.Realm)
				return
			}

			for _, kill := range gKills.Data {
				for _, ch := range channels {
					id := kill.KillID
					if w.subStore.IsKillProcessed(id, ch) {
						continue
					}

					mu.Lock()
					if _, ok := kills[id]; !ok {
						kills[id] = make(map[string]bool)
					}
					kills[id][ch] = true
					killRealms[id] = key.Realm
					mu.Unlock()
				}
			}
		}(key)
	}

	wg.Wait()
	return kills, killRealms
}

func (w *Worker) getPlayerKills(data map[storage.TrackedPlayerKey][]string) (map[int]map[string]bool, map[int]string) {
	kills := make(map[int]map[string]bool)
	killRealms := make(map[int]string)

	var mu sync.Mutex
	keys := make([]storage.TrackedPlayerKey, 0, len(data))
	for key := range data {
		keys = append(keys, key)
	}

	const workers = 6
	sem := make(chan struct{}, workers)
	var wg sync.WaitGroup

	for _, key := range keys {
		wg.Add(1)
		sem <- struct{}{}
		go func(key storage.TrackedPlayerKey) {
			defer wg.Done()
			defer func() { <-sem }()

			channels := data[key]
			w.apiLimiter.Wait()
			pKills, err := w.sirusClient.FetchPlayerLastActions(key.Realm, key.PlayerID)
			if err != nil || pKills == nil {
				w.logger.Error("[KillMonitor] pKills is nil or Fetch Player Latest BossKills err", "error", err, "player_id", key.PlayerID, "realm", key.Realm)
				return
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
					if w.subStore.IsKillProcessed(id, ch) {
						continue
					}

					mu.Lock()
					if _, ok := kills[id]; !ok {
						kills[id] = make(map[string]bool)
					}
					kills[id][ch] = true
					killRealms[id] = key.Realm
					mu.Unlock()
				}
			}
		}(key)
	}

	wg.Wait()
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

	data, err := os.ReadFile("mock/raid_run.json")
	if err != nil {
		w.logger.Error("Mock json read error", "error", err)
	}
	json.Unmarshal(data, &report)

	return report
}

func (w *Worker) makeMockMythicReport() *models.MythicReport {
	var report models.MythicReport

	data, err := os.ReadFile("mock/mythic_run.json")
	if err != nil {
		w.logger.Error("Mock json read error", "error", err)
	}
	json.Unmarshal(data, &report)

	return &report
}

func (w *Worker) sortKills(kills map[int]map[string]bool) []int {
	ids := make([]int, 0, len(kills))

	for id := range kills {
		ids = append(ids, id)
	}

	sort.Ints(ids)

	return ids
}
