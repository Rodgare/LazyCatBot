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

func KillMonitor(
	lbStore *storage.LeaderboardStorage,
	subStore *storage.SubscribeStorage,
	pSubStore *storage.PlayerSubscribeStorage,
	dg *discordgo.Session,
) {
	for {
		// discord.SendKillReport(dg, os.Getenv("DEBUG_CHANNEL_ID"), makeMockReport())
		guilds, err := subStore.GetTrackedGuilds()
		if err != nil {
			log.Printf("[KillMonitor] Getting tracked guilds err: %v", err)
			time.Sleep(1 * time.Minute)
			continue
		}

		players, err := pSubStore.GetTrackedPlayers()
		if err != nil {
			log.Printf("[KillMonitor] Getting tracked players err: %v", err)
			time.Sleep(1 * time.Minute)
			continue
		}

		kills := getKills(guilds, players, subStore)
		sortedKillsIDs := sortKills(kills)

		for _, killID := range sortedKillsIDs {
			enrichedKill, err := sirus.FetchBossFightDetails(killID)
			if err != nil {
				log.Printf("[KillMonitor] Fetch Boss Fight Details err: %v", err)
				time.Sleep(10 * time.Second)
				continue
			}

			report := createReport(enrichedKill, lbStore, killID)
			channels := kills[killID]

			for ch := range channels {
				discord.SendKillReport(dg, ch, report)
				subStore.MarkKillProcessed(killID, ch)
			}

			time.Sleep(2 * time.Second)
		}

		time.Sleep(1 * time.Minute)
	}
}

func sortKills(kills map[int]map[string]bool) []int {
	ids := make([]int, 0, len(kills))

	for id := range kills {
		ids = append(ids, id)
	}

	sort.Ints(ids)

	return ids
}

func getKills(guilds, players map[int][]string, subStore *storage.SubscribeStorage) map[int]map[string]bool {
	kills := make(map[int]map[string]bool)

	for gID, channels := range guilds {
		gKills, err := sirus.FetchGuildLatestBossKills(gID)
		if err != nil {
			log.Printf("[KillMonitor] Fetch Guild %d Latest BossKills err: %v", gID, err)
			continue
		}

		for _, kill := range gKills.Data {
			for _, ch := range channels {

				id := kill.KillID
				if sirus.IsGuildKillToday(kill.TimeEnd) && !subStore.IsKillProcessed(id, ch) {
					if _, ok := kills[id]; !ok {
						kills[id] = make(map[string]bool)
					}

					kills[id][ch] = true
				}
			}
		}
		time.Sleep(5 * time.Second)
	}

	for pID, channels := range players {
		pKills, err := sirus.FetchPlayerLastActions(pID)
		if err != nil {
			log.Printf("[KillMonitor] Fetch Player %d Latest BossKills err: %v", pID, err)
			continue
		}

		for _, kill := range *pKills {
			if kill.Type != "bosskill" {
				continue
			}
			for _, ch := range channels {
				id := kill.FightID

				if sirus.IsPlayerKillToday(kill.Date) && !subStore.IsKillProcessed(id, ch) {
					if _, ok := kills[id]; !ok {
						kills[id] = make(map[string]bool)
					}

					kills[id][ch] = true
				}
			}
		}
		time.Sleep(5 * time.Second)
	}

	return kills
}

func createReport(fight *sirus.BossFight, lbStore *storage.LeaderboardStorage, killID int) discord.BossKillReport {
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

		err := lbStore.UpsertPlayer(fight.Order, fight.Encounter, p)
		if err != nil {
			log.Printf("Upsert player in db error %v\n", err)
		}
		role := sirus.GetRoleString(p.ClassID, p.Spec)

		var specRank, specPrcnt, classRank, classPrcnt, ilvlRank, ilvlPrcnt, overallRank, overallPrcnt int
		if role == "dps" {
			specRank, specPrcnt, classRank, classPrcnt, ilvlRank, ilvlPrcnt, overallRank, overallPrcnt, err = lbStore.GetDpsRank(fight.Order, fight.Encounter, p)
		} else {
			specRank, specPrcnt, classRank, classPrcnt, ilvlRank, ilvlPrcnt, overallRank, overallPrcnt, err = lbStore.GetHpsRank(fight.Order, fight.Encounter, p)
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
