package worker

import (
	"LazyCatBot/internal/discord"
	"LazyCatBot/internal/sirus"
	"LazyCatBot/internal/storage"
	"log"
	"os"
	"time"

	"github.com/bwmarrin/discordgo"
)

func KillMonitor(
	lbStore *storage.LeaderboardStorage,
	subStore *storage.SubscribeStorage,
	dg *discordgo.Session,
) {
	var lastID int
	for {
		kills, err := sirus.FetchLatestBossKills()
		if err == nil && kills != nil {
			if len(kills.Data) > 0 {
				lastID = kills.Data[0].KillID
			}
			break
		}
		log.Printf("[KillMonitor] Failed to get initial ID: %v. Retrying in 10s...", err)
		time.Sleep(10 * time.Second)
	}

	for {
		kills, err := sirus.FetchLatestBossKills()
		if err != nil {
			log.Printf("[KillMonitor] FetchLatestBossKills error %v", err)
			time.Sleep(2 * time.Minute)
			continue
		}
		debugChannel := os.Getenv("DEBUG_CHANNEL_ID")
		newKills := getNewKills(kills, lastID)
		if len(newKills) > 0 {
			log.Printf("[KillMonitor] Poll found %d new kills (lastID: %d)", len(newKills), lastID)
		}

		for _, kill := range newKills {
			channels, _ := subStore.GetSubscribers(kill.GuildId)

			if debugChannel != "" {
				channels = append(channels, debugChannel)
			}

			if len(channels) == 0 {
				log.Printf("[KillMonitor] Skipping kill %d (Guild %d) - no subscribers", kill.KillID, kill.GuildId)
				continue
			}

			log.Printf("[KillMonitor] Fetching details for kill %d (Guild %d)...", kill.KillID, kill.GuildId)
			fight, err := sirus.FetchBossFightDetails(kill.KillID)
			time.Sleep(2 * time.Second)
			if err != nil {
				log.Printf("[KillMonitor] Error fetching details for kill %d: %v", kill.KillID, err)
				time.Sleep(2 * time.Second)
				continue
			}

			log.Printf("[KillMonitor] Successfully fetched details for [%s], creating report...", fight.Data.BossName)
			report := createReport(fight, lbStore)

			for _, ch := range channels {
				discord.SendKillReport(dg, ch, report)
				log.Printf("[KillMonitor] Sent report for [%s] (KillID %d) to channel %s", fight.Data.BossName, kill.KillID, ch)
			}
		}

		if len(kills.Data) > 0 {
			lastID = kills.Data[0].KillID
		}

		time.Sleep(2 * time.Minute)
	}
}

func createReport(fight *sirus.BossFight, lbStore *storage.LeaderboardStorage) discord.BossKillReport {
	totalDps := 0
	totalHps := 0
	for _, p := range fight.Data.Players {
		totalDps += p.Dps
		totalHps += p.Hps
	}

	report := discord.BossKillReport{
		BossName: fight.Data.BossName,
		Duration: fight.Data.FightLength,
		Attempts: fight.Data.Attempts,
		TotalDps: totalDps,
		TotalHps: totalHps,
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
