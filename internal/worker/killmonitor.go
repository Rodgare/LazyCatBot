package worker

import (
	"LazyCatBot/internal/config"
	"LazyCatBot/internal/models"
	"LazyCatBot/internal/sirus"
	"LazyCatBot/internal/storage"
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"os"
	"sort"
	"strings"
	"sync"
	"time"
)

type Reporter interface {
	SendKillReport(channelID string, report models.BossKillReport) error
	EditKillReport(channelID, messageID string, report models.BossKillReport) error
	SendMythicReport(channelID string, report *models.MythicReport) error
	EditMythicReport(channelID, messageID string, report *models.MythicReport) error
	SendMythicTopMock(channelID string) error
	GetChannelName(channelID string) string
	SendChannelNotice(channelID, text string) error
	SendQueueNotice(channelID, text string) (string, error)
	EditQueueNotice(channelID, messageID, text string) error
	DeleteNotice(channelID, messageID string) error
	SendAnnouncement(channelID, text string) error
	SendWeeklyDigest(channelID string, digest models.WeeklyDigest) error
}

// PremiumGate reports whether a Discord server has an active premium
// entitlement. Premium is guild-wide, so the channel is resolved to its server.
// A nil gate means every server is free.
type PremiumGate interface {
	IsPremium(guildID string) bool
}

// ChannelGuildResolver maps a Discord channel to its server id so the worker can
// apply the guild-wide premium check. An empty string means "unknown".
type ChannelGuildResolver interface {
	GetChannelGuildID(channelID string) string
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
	GetWelcomeChannel(discordID string) (string, error)
	GetDiscordIDByChannel(channelID string) (string, error)
	RemoveChannelSubscriptions(channelID string) error
	GetAllWelcomeChannels() ([]string, error)
	MetaGet(key string) (string, error)
	MetaSet(key, value string) error
}

func isMissingAccess(err error) bool {
	if err == nil {
		return false
	}
	msg := err.Error()
	return strings.Contains(msg, "Missing Access") || strings.Contains(msg, "50001")
}

const missingAccessSuppressInterval = 1800 // seconds (30 min)

func (w *Worker) isAccessSuppressed(channelID string) bool {
	now := int(time.Now().Unix())
	if last, ok := w.missingAccessSuppressed[channelID]; ok && now-last < missingAccessSuppressInterval {
		return true
	}
	return false
}

func (w *Worker) suppressAccess(channelID string) {
	w.missingAccessSuppressed[channelID] = int(time.Now().Unix())
}

func (w *Worker) notifyMissingAccess(channelID string) {
	discordID, _ := w.subStore.GetDiscordIDByChannel(channelID)
	welcomeID := ""
	if discordID != "" {
		welcomeID, _ = w.subStore.GetWelcomeChannel(discordID)
	}
	if welcomeID == "" {
		w.logger.Warn("[Processor] No welcome channel to report missing access", "channel", channelID)
		return
	}

	channelName := w.reporter.GetChannelName(channelID)
	text := fmt.Sprintf("⚠️ Не удалось отправить отчёт в канал **«%s»** (`%s`).\n"+
		"Похоже, у бота нет прав писать в этот канал. "+
		"Выдайте роли бота права **Просмотр канала** и **Отправка сообщений** в этом канале — "+
		"отчёты отправятся автоматически, ничего включать заново не нужно.", channelName, channelID)

	if err := w.reporter.SendChannelNotice(welcomeID, text); err != nil {
		w.logger.Error("[Processor] Send missing-access notice err", "error", err, "channel", channelID, "welcome_channel", welcomeID)
		return
	}
	w.logger.Warn("[Processor] Missing-access notice sent", "channel", channelID, "channel_name", channelName, "welcome_channel", welcomeID)
}

type SirusAPI interface {
	FetchBossFightDetails(realm string, fightID int) (*models.BossFight, error)
	FetchGuildLatestBossKills(realm string, guildID int) (*models.LatestBossKills, error)
	FetchPlayerLastActions(realm string, playerID int) (*models.PlayerLastActions, error)
	FetchGuildMembers(realm string, guildID int) (*[]models.GuildMembers, string, error)
	FetchCharacter(realm string, id int) (*models.CharacterData, error)
	FetchActualRaids(realm string) (models.ActualSirusRaids, error)
	FetchLeaderboard(realm string, raidID, bossID, classID, specID int, role string) ([]models.LeaderboardPlayer, bool, error)
	GetLatestMythicRuns(realm string) (*models.MythicRuns, error)
	FetchMythicRunDetails(realm string, runID int) (*models.MythicRun, error)
	FetchChallengeScores(realm string, season, weekID int) ([]models.MythicScorePlayer, int, error)
	GetSirusDates() (string, string)
}

type KillJob struct {
	KillID       int
	Realm        string
	Channels     map[string]bool
	Type         string
	MythicReport *models.MythicReport
	// JobID deduplicates repeated detections of the same kill.
	JobID string
	// Title and KillTime are surfaced in the queue notice when known.
	Title    string
	KillTime string
}

type queueNotice struct {
	ChannelID string
	MessageID string
	JobID     string
	Position  int
	Title     string
	KillTime  string
	CreatedAt time.Time
}

type Worker struct {
	sirusClient             SirusAPI
	lbStore                 *storage.LeaderboardStorage
	mythLbStore             *storage.MythicLeaderboardStorage
	killHistory             *storage.KillHistoryStorage
	subStore                SubscribeStore
	pSubStore               *storage.PlayerSubscribeStorage
	gmStore                 *storage.GuildMembersStorage
	charStore               *storage.CharacterCacheStorage
	arStore                 *storage.ActualRaidsStorage
	reporter                Reporter
	scheduler               *jobScheduler
	premiumGate             PremiumGate
	guildResolver           ChannelGuildResolver
	charPriority            chan PriorityChar
	apiLimiter              *RateLimiter
	killApiLimiter          *RateLimiter
	logger                  *slog.Logger
	cfg                     *config.Config
	missingAccessSuppressed map[string]int
	resyncMu                sync.Mutex
	noticeMu                sync.Mutex
	activeNotices           map[string][]queueNotice
	pendingJobs             map[string]struct{}
	deliveryMu              sync.Mutex
	delivered               map[string]struct{}
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
		sirusClient:             sc,
		lbStore:                 lb,
		subStore:                sub,
		pSubStore:               ps,
		gmStore:                 gm,
		charStore:               cc,
		arStore:                 ar,
		reporter:                reporter,
		scheduler:               newJobScheduler(),
		charPriority:            make(chan PriorityChar, 1000),
		apiLimiter:              NewRateLimiter(2, 1),
		killApiLimiter:          NewRateLimiter(2, 1),
		logger:                  logger,
		cfg:                     cfg,
		missingAccessSuppressed: make(map[string]int),
		activeNotices:           make(map[string][]queueNotice),
		pendingJobs:             make(map[string]struct{}),
		delivered:               make(map[string]struct{}),
	}
}

func (w *Worker) SetMythicLeaderboardStore(ml *storage.MythicLeaderboardStorage) {
	w.mythLbStore = ml
}

// SetKillHistoryStore wires persistence for attendance, digests and profiles.
func (w *Worker) SetKillHistoryStore(kh *storage.KillHistoryStorage) {
	w.killHistory = kh
}

// SetPremiumGate wires the premium entitlement lookup used to prioritize report
// delivery. Without it every channel is treated as free.
func (w *Worker) SetPremiumGate(g PremiumGate) {
	w.premiumGate = g
}

// SetGuildResolver wires channel-to-server resolution required by the
// guild-wide premium check.
func (w *Worker) SetGuildResolver(r ChannelGuildResolver) {
	w.guildResolver = r
}

// channelPremium reports whether the channel belongs to a premium server.
func (w *Worker) channelPremium(channelID string) bool {
	if w.premiumGate == nil || w.guildResolver == nil {
		return false
	}
	guildID := w.guildResolver.GetChannelGuildID(channelID)
	if guildID == "" {
		return false
	}
	return w.premiumGate.IsPremium(guildID)
}

// enqueueJob classifies a job by the channels it targets and pushes it into the
// priority scheduler. Free channels receive one queue notice per report; the
// same message is edited as the queue advances and removed when the report is
// generated. Repeated detections of the same kill are ignored while a job for
// it is still pending. Premium servers get their report without a placeholder
// because they never wait in the queue.
func (w *Worker) enqueueJob(job KillJob) {
	if job.JobID == "" {
		job.JobID = fmt.Sprintf("%s:%s:%d", job.Type, job.Realm, job.KillID)
	}

	w.noticeMu.Lock()
	if _, exists := w.pendingJobs[job.JobID]; exists {
		w.noticeMu.Unlock()
		return
	}
	w.pendingJobs[job.JobID] = struct{}{}
	w.noticeMu.Unlock()

	channels := make(map[string]bool, len(job.Channels))
	for ch := range job.Channels {
		if !w.isDelivered(job.JobID, ch) {
			channels[ch] = true
		}
	}
	if len(channels) == 0 {
		w.noticeMu.Lock()
		delete(w.pendingJobs, job.JobID)
		w.noticeMu.Unlock()
		return
	}
	job.Channels = channels

	freeChannels := make([]string, 0, len(job.Channels))
	hasPremium := false
	for ch := range job.Channels {
		if w.channelPremium(ch) {
			hasPremium = true
			continue
		}
		freeChannels = append(freeChannels, ch)
	}

	position := w.scheduler.freeLen() + 1
	if !hasPremium {
		for _, ch := range freeChannels {
			notice := queueNotice{
				ChannelID: ch,
				JobID:     job.JobID,
				Position:  position,
				Title:     job.Title,
				KillTime:  job.KillTime,
				CreatedAt: time.Now(),
			}
			msgID, err := w.reporter.SendQueueNotice(ch, notice.queuedText())
			if err != nil {
				w.logger.Warn("[Processor] Send queue notice err", "error", err, "channel", ch)
				continue
			}
			notice.MessageID = msgID
			w.noticeMu.Lock()
			w.activeNotices[job.JobID] = append(w.activeNotices[job.JobID], notice)
			w.noticeMu.Unlock()
		}
	}

	w.scheduler.push(job, hasPremium)
}

func (n queueNotice) queuedText() string {
	var b strings.Builder
	b.WriteString("⏳ Отчёт поставлен в очередь на генерацию.")
	if n.Title != "" {
		b.WriteString("\n**" + n.Title + "**")
	}
	if n.KillTime != "" {
		b.WriteString("\nКил: " + n.KillTime)
	}
	fmt.Fprintf(&b, "\nПозиция в очереди: **%d**.", n.Position)
	b.WriteString("\nСервер с **премиумом** получает отчёты без очереди. Подробнее — кнопка ниже.")
	return b.String()
}

func (n queueNotice) processingText() string {
	var b strings.Builder
	b.WriteString("⚙️ Генерируем отчёт…")
	if n.Title != "" {
		b.WriteString("\n**" + n.Title + "**")
	}
	if n.KillTime != "" {
		b.WriteString("\nКил: " + n.KillTime)
	}
	return b.String()
}

// markJobProcessing updates the job's notices to the "generating" state.
func (w *Worker) markJobProcessing(job KillJob) {
	w.noticeMu.Lock()
	notices := append([]queueNotice(nil), w.activeNotices[job.JobID]...)
	w.noticeMu.Unlock()

	for _, n := range notices {
		if err := w.reporter.EditQueueNotice(n.ChannelID, n.MessageID, n.processingText()); err != nil {
			w.logger.Debug("[Processor] Edit queue notice err", "error", err, "channel", n.ChannelID)
		}
	}
}

// editJobNotices replaces the text of every active notice for a job.
func (w *Worker) editJobNotices(jobID, text string) {
	w.noticeMu.Lock()
	notices := append([]queueNotice(nil), w.activeNotices[jobID]...)
	w.noticeMu.Unlock()

	for _, n := range notices {
		if err := w.reporter.EditQueueNotice(n.ChannelID, n.MessageID, text); err != nil {
			w.logger.Debug("[Processor] Edit queue notice err", "error", err, "channel", n.ChannelID)
		}
	}
}

// refreshNoticePositions rewrites the queue position on every waiting notice.
func (w *Worker) refreshNoticePositions() {
	ids := w.scheduler.freeJobIDs()
	positions := make(map[string]int, len(ids))
	for i, id := range ids {
		positions[id] = i + 1
	}

	w.noticeMu.Lock()
	var updates []queueNotice
	for jobID, notices := range w.activeNotices {
		position, ok := positions[jobID]
		if !ok {
			continue
		}
		for i := range notices {
			if notices[i].Position == position {
				continue
			}
			notices[i].Position = position
			updates = append(updates, notices[i])
		}
	}
	w.noticeMu.Unlock()

	for _, n := range updates {
		if err := w.reporter.EditQueueNotice(n.ChannelID, n.MessageID, n.queuedText()); err != nil {
			w.logger.Debug("[Processor] Edit queue position err", "error", err, "channel", n.ChannelID)
		}
	}
}

// finishJob deletes the job's queue notices and lets a future poll retry the
// kill if it was not marked as processed.
func (w *Worker) finishJob(job KillJob) {
	w.noticeMu.Lock()
	notices := w.activeNotices[job.JobID]
	delete(w.activeNotices, job.JobID)
	delete(w.pendingJobs, job.JobID)
	w.noticeMu.Unlock()

	for _, n := range notices {
		if n.MessageID == "" {
			continue
		}
		if err := w.reporter.DeleteNotice(n.ChannelID, n.MessageID); err != nil {
			w.logger.Debug("[Processor] Delete queue notice err", "error", err, "channel", n.ChannelID)
		}
	}
	w.refreshNoticePositions()
}

func deliveryKey(jobID, channelID string) string {
	return jobID + ":" + channelID
}

func (w *Worker) isDelivered(jobID, channelID string) bool {
	w.deliveryMu.Lock()
	defer w.deliveryMu.Unlock()
	_, ok := w.delivered[deliveryKey(jobID, channelID)]
	return ok
}

func (w *Worker) markDelivered(jobID, channelID string) {
	w.deliveryMu.Lock()
	w.delivered[deliveryKey(jobID, channelID)] = struct{}{}
	w.deliveryMu.Unlock()
}

// takeNotice removes and returns the notice for a channel so finishJob will not
// delete a message that is about to be edited into the report.
func (w *Worker) takeNotice(jobID, channelID string) (queueNotice, bool) {
	w.noticeMu.Lock()
	defer w.noticeMu.Unlock()

	notices := w.activeNotices[jobID]
	for i, n := range notices {
		if n.ChannelID != channelID {
			continue
		}
		w.activeNotices[jobID] = append(notices[:i], notices[i+1:]...)
		if len(w.activeNotices[jobID]) == 0 {
			delete(w.activeNotices, jobID)
		}
		return n, true
	}
	return queueNotice{}, false
}

const killPollWindowSeconds = 90

const killPollWindow = killPollWindowSeconds * time.Second

func (w *Worker) prepareGuildPollLimiter(guildCount int) {
	if guildCount <= 1 {
		w.killApiLimiter = NewRateLimiter(1000, 1)
		return
	}
	rate := guildCount / killPollWindowSeconds
	if rate < 1 {
		rate = 1
	} else if rate > 4 {
		rate = 4
	}
	w.killApiLimiter = NewRateLimiter(rate, 1)
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

		started := time.Now()

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

		w.prepareGuildPollLimiter(len(guilds))

		kills, killRealms, killTimes := w.getGuildKills(guilds)
		sortedKillsIDs := w.sortKills(kills)

		for _, killID := range sortedKillsIDs {
			select {
			case <-ctx.Done():
				return
			default:
			}
			w.enqueueJob(KillJob{
				KillID:   killID,
				Realm:    killRealms[killID],
				Channels: kills[killID],
				Type:     "raid",
				KillTime: killTimes[killID],
			})
		}

		elapsed := time.Since(started)
		remain := killPollWindow - elapsed
		if remain > 0 {
			select {
			case <-ctx.Done():
				w.logger.Info("[KillMonitor] Guild monitor stopped")
				return
			case <-time.After(remain):
			}
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

		kills, killRealms, killTimes := w.getPlayerKills(players)
		sortedKillsIDs := w.sortKills(kills)

		for _, killID := range sortedKillsIDs {
			select {
			case <-ctx.Done():
				return
			default:
			}
			w.enqueueJob(KillJob{
				KillID:   killID,
				Realm:    killRealms[killID],
				Channels: kills[killID],
				Type:     "raid",
				KillTime: killTimes[killID],
			})
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
		w.killApiLimiter.Wait()
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
			default:
			}
			title := report.Name
			if title != "" && report.KeystoneLevel > 0 {
				title = fmt.Sprintf("%s +%d", title, report.KeystoneLevel)
			}
			killTime := ""
			if report.DateTime > 0 {
				killTime = time.Unix(int64(report.DateTime), 0).In(time.FixedZone("MSK", 3*3600)).Format("02.01 15:04")
			}
			w.enqueueJob(KillJob{
				KillID:       runID,
				Realm:        realm,
				Channels:     channels,
				MythicReport: &report,
				Type:         "myth",
				Title:        title,
				KillTime:     killTime,
			})
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
		job, ok := w.scheduler.pop(ctx)
		if !ok {
			w.logger.Info("[KillMonitor] Processor stopped")
			return
		}

		w.markJobProcessing(job)
		w.refreshNoticePositions()

		queued := w.scheduler.total()
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

func (w *Worker) processRaids(job KillJob) {
	defer w.finishJob(job)
	var report *models.BossKillReport

	for ch := range job.Channels {
		if w.isDelivered(job.JobID, ch) || w.subStore.IsKillProcessed(job.KillID, ch) {
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

			notice := "⚙️ Генерируем отчёт…"
			if rep.BossName != "" {
				notice += "\n**" + rep.BossName + "**"
			}
			if rep.MapName != "" {
				notice += " · " + rep.MapName
			}
			if job.KillTime != "" {
				notice += "\nКил: " + job.KillTime
			}
			w.editJobNotices(job.JobID, notice)

			if w.killHistory != nil {
				if err := w.killHistory.RecordKill(rep); err != nil {
					w.logger.Error("[Processor] Record kill history err", "error", err, "kill_id", job.KillID)
				}
			}

			for _, p := range enrichedKill.Data.Players {
				w.enqueueCharRefreshIfStale(job.Realm, p.Name, p.GUID)
			}
		}

		if !w.subStore.IsReportsEnabled(report.GuildID, ch) {
			w.markDelivered(job.JobID, ch)
			w.markProcessed(job.KillID, ch)
			continue
		}
		if w.isAccessSuppressed(ch) {
			continue
		}

		notice, hasNotice := w.takeNotice(job.JobID, ch)
		w.markDelivered(job.JobID, ch)
		w.markProcessed(job.KillID, ch)
		w.scheduleKillReport(job, ch, notice, hasNotice, *report)
	}
}

func (w *Worker) markProcessed(killID int, ch string) {
	if err := w.subStore.MarkKillProcessed(killID, ch); err != nil {
		w.logger.Error("[Processor] Mark kill processed err", "error", err, "kill_id", killID, "channel", ch)
	}
}

// queueNoticeMinVisible keeps the queue message on screen long enough that a
// free-channel report never looks like a flicker.
const queueNoticeMinVisible = 12 * time.Second

func (w *Worker) scheduleKillReport(job KillJob, ch string, notice queueNotice, hasNotice bool, report models.BossKillReport) {
	if !hasNotice {
		w.deliverKillReport(job, ch, "", false, report)
		return
	}
	go func() {
		if wait := time.Until(notice.CreatedAt.Add(queueNoticeMinVisible)); wait > 0 {
			time.Sleep(wait)
		}
		w.deliverKillReport(job, ch, notice.MessageID, true, report)
	}()
}

func (w *Worker) deliverKillReport(job KillJob, ch, messageID string, edit bool, report models.BossKillReport) {
	var err error
	if edit {
		err = w.reporter.EditKillReport(ch, messageID, report)
		if err != nil {
			w.logger.Warn("[Processor] Edit kill report err, sending a new message",
				"error", err, "kill_id", job.KillID, "channel", ch)
			_ = w.reporter.DeleteNotice(ch, messageID)
			err = w.reporter.SendKillReport(ch, report)
		}
	} else {
		err = w.reporter.SendKillReport(ch, report)
	}
	if err != nil {
		w.handleReportDeliveryError(job.KillID, ch, err)
	}
}

func (w *Worker) handleReportDeliveryError(killID int, ch string, err error) {
	if isMissingAccess(err) {
		w.suppressAccess(ch)
		w.notifyMissingAccess(ch)
		if rmErr := w.subStore.RemoveChannelSubscriptions(ch); rmErr != nil {
			w.logger.Error("[Processor] Remove inaccessible channel subscriptions err", "error", rmErr, "channel", ch)
		} else {
			w.logger.Warn("[Processor] Removed subscriptions for inaccessible channel", "channel", ch)
		}
		w.logger.Warn("[Processor] Missing access for channel (subscriptions removed)", "error", err, "kill_id", killID, "channel", ch)
		return
	}
	w.logger.Error("[Processor] Send kill report err", "error", err, "kill_id", killID, "channel", ch)
}

func (w *Worker) processMythicRuns(job KillJob) {
	defer w.finishJob(job)
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
		if w.isDelivered(job.JobID, ch) || w.subStore.IsKillProcessed(job.KillID, ch) {
			continue
		}

		if !w.subStore.IsMythicReportsEnabled(ch) {
			w.markDelivered(job.JobID, ch)
			w.markProcessed(job.KillID, ch)
			continue
		}
		if w.isAccessSuppressed(ch) {
			continue
		}

		notice, hasNotice := w.takeNotice(job.JobID, ch)
		w.markDelivered(job.JobID, ch)
		w.markProcessed(job.KillID, ch)
		w.scheduleMythicReport(job, ch, notice, hasNotice, job.MythicReport)
	}

}

func (w *Worker) scheduleMythicReport(job KillJob, ch string, notice queueNotice, hasNotice bool, report *models.MythicReport) {
	if !hasNotice {
		w.deliverMythicReport(job, ch, "", false, report)
		return
	}
	go func() {
		if wait := time.Until(notice.CreatedAt.Add(queueNoticeMinVisible)); wait > 0 {
			time.Sleep(wait)
		}
		w.deliverMythicReport(job, ch, notice.MessageID, true, report)
	}()
}

func (w *Worker) deliverMythicReport(job KillJob, ch, messageID string, edit bool, report *models.MythicReport) {
	var err error
	if edit {
		err = w.reporter.EditMythicReport(ch, messageID, report)
		if err != nil {
			w.logger.Warn("[Processor] Edit mythic report err, sending a new message",
				"error", err, "run_id", job.KillID, "channel", ch)
			_ = w.reporter.DeleteNotice(ch, messageID)
			err = w.reporter.SendMythicReport(ch, report)
		}
	} else {
		err = w.reporter.SendMythicReport(ch, report)
	}
	if err != nil {
		w.handleReportDeliveryError(job.KillID, ch, err)
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
				job.Realm, m.ClassID, m.SpecID, m.MythicRating,
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

func (w *Worker) getGuildKills(data map[storage.TrackedGuildKey][]string) (map[int]map[string]bool, map[int]string, map[int]string) {
	kills := make(map[int]map[string]bool)
	killRealms := make(map[int]string)
	killTimes := make(map[int]string)

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
			w.killApiLimiter.Wait()
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
					killTimes[id] = kill.TimeEnd
					mu.Unlock()
				}
			}
		}(key)
	}

	wg.Wait()
	return kills, killRealms, killTimes
}

func (w *Worker) getPlayerKills(data map[storage.TrackedPlayerKey][]string) (map[int]map[string]bool, map[int]string, map[int]string) {
	kills := make(map[int]map[string]bool)
	killRealms := make(map[int]string)
	killTimes := make(map[int]string)

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
					id := kill.FightIDValue()
					if w.subStore.IsKillProcessed(id, ch) {
						continue
					}

					mu.Lock()
					if _, ok := kills[id]; !ok {
						kills[id] = make(map[string]bool)
					}
					kills[id][ch] = true
					killRealms[id] = key.Realm
					killTimes[id] = kill.Date
					mu.Unlock()
				}
			}
		}(key)
	}

	wg.Wait()
	return kills, killRealms, killTimes
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
		BossID:    fight.Encounter,
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
		setPieces := sirus.GetSetPieces(p.Itemset)
		role := sirus.GetRoleString(p.ClassID, p.Spec)

		err := w.lbStore.UpsertPlayer(realm, fight.Order, fight.Encounter, p)
		if err != nil {
			w.logger.Error("Upsert player in db error", "error", err)
		}

		playerReport := models.PlayerReport{
			Name:      p.Name,
			GUID:      p.GUID,
			Dps:       p.Dps,
			Hps:       p.Hps,
			Ilvl:      p.Ilvl,
			SpecID:    p.Spec,
			ClassID:   p.ClassID,
			SpecName:  specName,
			T4:        t4Count,
			SetPieces: setPieces,
			Role:      sirus.GetRole(p.ClassID, p.Spec),
			Zodiac:    p.Zodiac.ID,
			Category:  p.Category,
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
