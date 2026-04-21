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
		kills, err := sirus.FetchLatestBossKills(1)
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
		var newKills []sirus.BossKill
		page := 1
		maxPages := 10

		for page <= maxPages {
			kills, err := sirus.FetchLatestBossKills(page)
			if err != nil {
				log.Printf("[KillMonitor] FetchLatestBossKills page %d error %v", page, err)
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
			time.Sleep(2 * time.Second)
		}

		if len(newKills) == 0 {
			time.Sleep(30 * time.Second)
			continue
		}

		maxNewID := lastID
		for _, kill := range newKills {
			if kill.KillID > maxNewID {
				maxNewID = kill.KillID
			}
		}

		debugChannel := os.Getenv("DEBUG_CHANNEL_ID")
		log.Printf("[KillMonitor] Poll found %d new kills (lastID: %d -> %d)", len(newKills), lastID, maxNewID)

		for i := len(newKills) - 1; i >= 0; i-- {
			kill := newKills[i]
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
			report := createReport(fight, lbStore, kill.KillID)

			for _, ch := range channels {
				discord.SendKillReport(dg, ch, report)
				log.Printf("[KillMonitor] Sent report for [%s] (KillID %d) to channel %s", fight.Data.BossName, kill.KillID, ch)
			}
		}

		lastID = maxNewID
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
