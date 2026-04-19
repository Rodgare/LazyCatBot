package worker

import (
	"LazyCatBot/internal/discord"
	"LazyCatBot/internal/sirus"
	"LazyCatBot/internal/storage"
	"log"
	"slices"
	"time"

	"github.com/bwmarrin/discordgo"
)

var GuildsIDsToWatch = []int{913}

func KillMonitor(
	lbStore *storage.LeaderboardStorage,
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

		newKills := getNewKills(kills, lastID)
		guildKills := getGuildKills(newKills, GuildsIDsToWatch)
		killsDetails := getKillsDetails(guildKills)
		killsReports := getKillsReports(killsDetails, lbStore)

		for _, report := range killsReports {
			discord.SendKillReport(dg, "700024788164411435", report)
		}

		if len(kills.Data) > 0 {
			lastID = kills.Data[0].KillID
		}

		time.Sleep(2 * time.Minute)
	}
}

func getKillsReports(kills []sirus.BossFight, lbStore *storage.LeaderboardStorage) []discord.BossKillReport {
	var bossKillReports []discord.BossKillReport

	for _, kill := range kills {
		report := discord.BossKillReport{
			BossName: kill.Data.BossName,
		}

		for _, player := range kill.Data.Players {
			playerReport := discord.PlayerReport{
				Name:     player.Name,
				Dps:      player.Dps,
				SpecRank: lbStore.GetSpecRank(kill.Order, kill.Encounter, player.ClassID, player.Spec, player.Dps),
			}

			report.Players = append(report.Players, playerReport)
		}

		bossKillReports = append(bossKillReports, report)
	}

	return bossKillReports
}

func getKillsDetails(kills []sirus.BossKill) []sirus.BossFight {
	var fights []sirus.BossFight

	for _, kill := range kills {
		fight, err := sirus.FetchBossFightDetails(kill.KillID)
		if err != nil {
			log.Printf("[KillMonitor] FetchBossFightDetails error %v", err)
			time.Sleep(2 * time.Second)
			continue
		}
		fights = append(fights, *fight)
		time.Sleep(2 * time.Second)
	}

	return fights
}

func getGuildKills(kills []sirus.BossKill, guildIDs []int) []sirus.BossKill {
	var guildKills []sirus.BossKill

	for _, kill := range kills {
		if slices.Contains(guildIDs, kill.GuildId) {
			guildKills = append(guildKills, kill)
		}
	}

	return guildKills
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
