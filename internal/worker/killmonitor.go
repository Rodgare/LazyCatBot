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
	chSubStore *storage.CharacterSubscribeStorage,
	dg *discordgo.Session,
) {
	var lastKillID int

	for {
		// discord.SendKillReport(dg, os.Getenv("DEBUG_CHANNEL_ID"), makeMockReport())

		guilds, err := subStore.GetTrackedGuilds()
		if err != nil {
			log.Fatalf("[KillMonitor] Getting tracked guilds err: %v", err)
		}

		characters, err := chSubStore.GetTrackedCharacters()
		if err != nil {
			log.Fatalf("[KillMonitor] Getting tracked characters err: %v", err)
		}

		kills := getKills(guilds, characters, lastKillID)
		sortedKillsIDs := sortKills(kills)

		if lastKillID == 0 {
			for id := range kills {
				lastKillID = max(lastKillID, id)
			}
			continue
		}

		for _, killID := range sortedKillsIDs {
			enrichedKill, err := sirus.FetchBossFightDetails(killID)
			if err != nil {
				log.Printf("[KillMonitor] Fetch Boss Fight Details err: %v", err)
				time.Sleep(1 * time.Minute)
				continue
			}

			report := createReport(enrichedKill, lbStore, killID)
			channels := kills[killID]

			for ch := range channels {
				discord.SendKillReport(dg, ch, report)
			}

			lastKillID = max(lastKillID, killID)
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

func getKills(guilds, players map[int][]string, lastID int) map[int]map[string]bool {
	// [killID][channel]true
	kills := make(map[int]map[string]bool, len(guilds)+len(players))

	for gID, channels := range guilds {
		gKills, err := sirus.FetchGuildLatestBossKills(gID)
		if err != nil {
			log.Printf("[KillMonitor] Fetch Guild Latest BossKills err: %v", err)
			time.Sleep(5 * time.Second)
			continue
		}

		for _, kill := range gKills.Data {
			if kill.KillID > lastID {
				if _, ok := kills[kill.KillID]; !ok {
					kills[kill.KillID] = make(map[string]bool)
				}

				for _, ch := range channels {
					kills[kill.KillID][ch] = true
				}
			}
		}

		time.Sleep(5 * time.Second)
	}

	for pID, channels := range players {
		pKills, err := sirus.FetchPlayerLatestBossKills(pID)
		if err != nil {
			log.Printf("[KillMonitor] Fetch Player Latest BossKills err: %v", err)
			time.Sleep(5 * time.Second)
			continue
		}

		for _, kill := range pKills.Data {
			if kill.ID > lastID {
				if _, ok := kills[kill.ID]; !ok {
					kills[kill.ID] = make(map[string]bool)
				}

				for _, ch := range channels {
					kills[kill.ID][ch] = true
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

		specRank, specPrcnt, classRank, classPrcnt, ilvlRank, ilvlPrcnt, overallRank, overallPrcnt, err := lbStore.GetRank(fight.Order, fight.Encounter, p)
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
