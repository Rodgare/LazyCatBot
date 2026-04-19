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
		log.Printf("[KillMonitor] Не удалось получить стартовый ID: %v. Пробую снова через 10 сек...", err)
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

		for _, kill := range newKills {
			channels, _ := subStore.GetSubscribers(kill.GuildId)

			if debugChannel != "" {
				channels = append(channels, debugChannel)
			}

			if len(channels) == 0 {
				continue
			}

			fight, err := sirus.FetchBossFightDetails(kill.KillID)
			time.Sleep(2 * time.Second)
			if err != nil {
				time.Sleep(2 * time.Second)
				continue
			}

			report := createReport(fight, lbStore)

			for _, ch := range channels {
				discord.SendKillReport(dg, ch, report)
			}
		}

		if len(kills.Data) > 0 {
			lastID = kills.Data[0].KillID
		}

		time.Sleep(2 * time.Minute)
	}
}

func createReport(fight *sirus.BossFight, lbStore *storage.LeaderboardStorage) discord.BossKillReport {
	report := discord.BossKillReport{
		BossName: fight.Data.BossName,
	}

	for _, player := range fight.Data.Players {
		playerReport := discord.PlayerReport{
			Name: player.Name,
			Dps:  player.Dps,
			SpecRank: lbStore.GetSpecRank(fight.Order, fight.Encounter, player.ClassID,
				player.Spec, player.Dps),
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
