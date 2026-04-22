package worker

import (
	"LazyCatBot/internal/discord"
	"LazyCatBot/internal/sirus"
	"LazyCatBot/internal/storage"
	"fmt"
	"log"
	"time"

	"github.com/bwmarrin/discordgo"
)

func KillMonitor(
	lbStore *storage.LeaderboardStorage,
	subStore *storage.SubscribeStorage,
	dg *discordgo.Session,
) {
	guildLastKills := make(map[int]int)

	for {
		guilds, err := subStore.GetTrackedGuilds()
		if err != nil {
			log.Printf("[KillMonitor] Error getting tracked guilds: %v\n", err)
			time.Sleep(30 * time.Second)
			continue
		}

		// debugChannel := os.Getenv("DEBUG_CHANNEL_ID")

		for _, guildID := range guilds {
			lastID := guildLastKills[guildID]
			if lastID == 0 {
				kills, err := sirus.FetchGuildLatestBossKills(1, guildID)
				if err == nil && kills != nil && len(kills.Data) > 0 {
					guildLastKills[guildID] = kills.Data[0].KillID
					fmt.Printf("[KillMonitor] Initialized tracking for guild %d (LastID: %d)\n", guildID, kills.Data[0].KillID)
				}
				continue
			}

			var newKills []sirus.BossKill
			page := 1
			maxPages := 5

			for page <= maxPages {
				kills, err := sirus.FetchGuildLatestBossKills(page, guildID)
				if err != nil {
					log.Printf("[KillMonitor] FetchGuildLatestBossKills (Guild: %d, Page: %d) error: %v\n", guildID, page, err)
					break
				}

				if len(kills.Data) == 0 {
					break
				}

				pageNewKills := getNewKills(kills, lastID)
				newKills = append(newKills, pageNewKills...)

				oldestOnPage := kills.Data[len(kills.Data)-1].KillID
				if oldestOnPage <= lastID {
					break
				}

				page++
				time.Sleep(1 * time.Second)
			}

			if len(newKills) == 0 {
				continue
			}

			maxNewID := lastID
			for _, kill := range newKills {
				if kill.KillID > maxNewID {
					maxNewID = kill.KillID
				}
			}

			fmt.Printf("[KillMonitor] Guild %d found %d new kills (lastID: %d -> %d)\n", guildID, len(newKills), lastID, maxNewID)

			for i := len(newKills) - 1; i >= 0; i-- {
				kill := newKills[i]
				channels, _ := subStore.GetSubscribers(kill.GuildId)

				// if debugChannel != "" {
				// 	channels = append(channels, debugChannel)
				// }

				if len(channels) == 0 {
					continue
				}

				log.Printf("[KillMonitor] Fetching details for kill %d (Guild %d)...\n", kill.KillID, kill.GuildId)
				fight, err := sirus.FetchBossFightDetails(kill.KillID)
				time.Sleep(2 * time.Second)
				if err != nil {
					log.Printf("[KillMonitor] Error fetching details for kill %d: %v\n", kill.KillID, err)
					continue
				}

				log.Printf("[KillMonitor] Successfully fetched details for [%s], creating report...\n", fight.Data.BossName)
				report := createReport(fight, lbStore, kill.KillID)

				for _, ch := range channels {
					discord.SendKillReport(dg, ch, report)
					log.Printf("[KillMonitor] Sent report for [%s] (KillID %d) to channel %s\n", fight.Data.BossName, kill.KillID, ch)
				}
			}

			guildLastKills[guildID] = maxNewID
			time.Sleep(2 * time.Second)
		}

		time.Sleep(30 * time.Second)
	}
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
		playerReport := discord.PlayerReport{
			Name:     p.Name,
			Dps:      p.Dps,
			Hps:      p.Hps,
			Ilvl:     p.Ilvl,
			ClassID:  p.ClassID,
			Role:     sirus.GetRole(p.ClassID, p.Spec),
			SpecName: specName,
			SpecRank: lbStore.GetSpecRank(fight.Order, fight.Encounter, p.ClassID,
				p.Spec, p.Dps),
		}
		report.Players = append(report.Players, playerReport)
	}

	return report
}

func getNewKills(kills *sirus.LatestBossKills, lastID int) []sirus.BossKill {
	var newKills []sirus.BossKill

	for _, kill := range kills.Data {
		if kill.KillID > lastID {
			newKills = append(newKills, kill)
		}
	}

	return newKills
}
