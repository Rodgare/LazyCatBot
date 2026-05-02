package worker

import (
	"LazyCatBot/internal/discord"
	"LazyCatBot/internal/sirus"
	"LazyCatBot/internal/storage"
	"encoding/json"
	"log"
	"os"
	"sort"
	"time"

	"github.com/bwmarrin/discordgo"
)

type KillJob struct {
	KillID   int
	Channels map[string]bool
}

type Worker struct {
	lbStore   *storage.LeaderboardStorage
	subStore  *storage.SubscribeStorage
	pSubStore *storage.PlayerSubscribeStorage
	dg        *discordgo.Session
	killQueue chan KillJob
}

func NewWorker(lb *storage.LeaderboardStorage,
	sub *storage.SubscribeStorage,
	ps *storage.PlayerSubscribeStorage,
	dg *discordgo.Session) *Worker {
	return &Worker{
		lbStore:   lb,
		subStore:  sub,
		pSubStore: ps,
		dg:        dg,
		killQueue: make(chan KillJob, 100),
	}
}

func (w *Worker) GuildKillMonitor() {
	for {
		// discord.SendKillReport(dg, os.Getenv("DEBUG_CHANNEL_ID"), makeMockReport())
		guilds, err := w.subStore.GetTrackedGuilds()
		if err != nil {
			log.Printf("[KillMonitor] Getting tracked guilds err: %v", err)
			time.Sleep(1 * time.Minute)
			continue
		}

		kills := w.getKills(guilds, true)
		sortedKillsIDs := sortKills(kills)

		for _, killID := range sortedKillsIDs {
			w.killQueue <- KillJob{
				KillID:   killID,
				Channels: kills[killID],
			}
		}

		time.Sleep(1 * time.Minute)
	}

}

func (w *Worker) PlayerKillMonitor() {
	for {
		// discord.SendKillReport(dg, os.Getenv("DEBUG_CHANNEL_ID"), makeMockReport())
		players, err := w.pSubStore.GetTrackedPlayers()
		if err != nil {
			log.Printf("[KillMonitor] Getting tracked players err: %v", err)
			time.Sleep(1 * time.Minute)
			continue
		}

		kills := w.getKills(players, false)
		sortedKillsIDs := sortKills(kills)

		for _, killID := range sortedKillsIDs {
			w.killQueue <- KillJob{
				KillID:   killID,
				Channels: kills[killID],
			}
		}

		time.Sleep(1 * time.Minute)
	}
}

func (w *Worker) StartProcessor() {
	for job := range w.killQueue {
		var report *discord.BossKillReport

		for ch := range job.Channels {
			if w.subStore.IsKillProcessed(job.KillID, ch) {
				continue
			}

			if report == nil {
				enrichedKill, err := sirus.FetchBossFightDetails(job.KillID)
				if err != nil {
					log.Printf("[Processor] Fetch Boss Fight Details err: %v", err)
					break
				}

				rep := w.createReport(enrichedKill, job.KillID)
				report = &rep
			}

			discord.SendKillReport(w.dg, ch, *report)
			w.subStore.MarkKillProcessed(job.KillID, ch)

			time.Sleep(300 * time.Millisecond)
		}
	}
}

func (w *Worker) getKills(data map[int][]string, isGuild bool) map[int]map[string]bool {
	kills := make(map[int]map[string]bool)

	if isGuild {
		for gID, channels := range data {
			gKills, err := sirus.FetchGuildLatestBossKills(gID)
			if err != nil {
				log.Printf("[KillMonitor] Fetch Guild %d Latest BossKills err: %v", gID, err)
				continue
			}

			for _, kill := range gKills.Data {
				for _, ch := range channels {
					// fmt.Printf("for kills killID %d\n", kill.KillID)

					id := kill.KillID
					if sirus.IsGuildKillToday(kill.TimeEnd) && !w.subStore.IsKillProcessed(id, ch) {
						if _, ok := kills[id]; !ok {
							kills[id] = make(map[string]bool)
						}

						kills[id][ch] = true
					}
				}
			}
			time.Sleep(300 * time.Millisecond)
		}
	} else {
		for pID, channels := range data {
			pKills, err := sirus.FetchPlayerLastActions(pID)
			if err != nil || pKills == nil {
				log.Printf("[KillMonitor] pKills is nil or Fetch Player %d Latest BossKills err: %v", pID, err)
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
					}
				}
			}
			time.Sleep(300 * time.Millisecond)
		}
	}

	return kills
}

func (w *Worker) createReport(fight *sirus.BossFight, killID int) discord.BossKillReport {
	totalDps := 0
	totalHps := 0
	for _, p := range fight.Data.Players {
		totalDps += p.Dps
		totalHps += p.Hps
	}

	report := discord.BossKillReport{
		MapName:   fight.Data.MapName,
		BossName:  fight.Data.BossName,
		RaidOrder: fight.Order,
		KillID:    killID,
		Duration:  fight.Data.FightLength,
		Attempts:  fight.Data.Attempts,
		KilledAt:  fight.Data.KilledAt,
		GuildID:   fight.Data.Guild.ID,
		GuildName: fight.Data.Guild.Name,
		TotalDps:  totalDps,
		TotalHps:  totalHps,
	}

	for _, loot := range fight.Data.Loots {
		lootReport := discord.LootReport{
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

		err := w.lbStore.UpsertPlayer(fight.Order, fight.Encounter, p, role)
		if err != nil {
			log.Printf("Upsert player in db error %v\n", err)
		}

		var specRank, specPrcnt, classRank, classPrcnt, ilvlRank, ilvlPrcnt, overallRank, overallPrcnt int
		if role == "dps" {
			specRank, specPrcnt, classRank, classPrcnt, ilvlRank, ilvlPrcnt, overallRank, overallPrcnt, err = w.lbStore.GetDpsRank(fight.Order, fight.Encounter, p)
		} else {
			specRank, specPrcnt, classRank, classPrcnt, ilvlRank, ilvlPrcnt, overallRank, overallPrcnt, err = w.lbStore.GetHpsRank(fight.Order, fight.Encounter, p)
		}
		if err != nil {
			log.Printf("Get player rank error %v\n", err)
			continue
		}

		playerReport := discord.PlayerReport{
			Name:              p.Name,
			Dps:               p.Dps,
			Hps:               p.Hps,
			Ilvl:              p.Ilvl,
			ClassID:           p.ClassID,
			Role:              sirus.GetRole(p.ClassID, p.Spec),
			SpecName:          specName,
			SpecID:            p.Spec,
			T4:                t4Count,
			SpecRank:          specRank,
			SpecPercentile:    specPrcnt,
			ClassRank:         classRank,
			ClassPercentile:   classPrcnt,
			IlvlRank:          ilvlRank,
			IlvlPercentile:    ilvlPrcnt,
			OverallRank:       overallRank,
			OverallPercentile: overallPrcnt,
			Zodiac:            p.Zodiac.ID,
			Category:          p.Category,
		}
		report.Players = append(report.Players, playerReport)
	}

	return report
}

func makeMockReport() discord.BossKillReport {
	var report discord.BossKillReport

	data, _ := os.ReadFile("internal/sirus/testdata/mock_sirus_boss_fight.json")
	json.Unmarshal(data, &report)

	return report
}

func sortKills(kills map[int]map[string]bool) []int {
	ids := make([]int, 0, len(kills))

	for id := range kills {
		ids = append(ids, id)
	}

	sort.Ints(ids)

	return ids
}
