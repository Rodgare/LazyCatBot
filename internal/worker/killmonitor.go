package worker

import (
	"LazyCatBot/internal/sirus"
	"LazyCatBot/internal/storage"
	"fmt"
	"log"
	"slices"
	"time"
)

var GuildsIDsToWatch = []int{913}

func KillMonitor(store *storage.BossKillsStorage) {
	kills, err := sirus.FetchLatestBossKills()
	if err != nil {
		log.Printf("[KillMonitor] FetchGuildKills error %v", err)
	}
	lastID := 0
	if len(kills.Data) > 0 {
		lastID = kills.Data[0].KillID
	}

	for {
		kills, err := sirus.FetchLatestBossKills()
		if err != nil {
			log.Printf("[KillMonitor] FetchLatestBossKills error %v", err)
		}

		newKills := getNewKills(kills, lastID)
		guildKills := getGuildKills(newKills, GuildsIDsToWatch)
		killsDetails := getKillsDetails(guildKills)

		fmt.Printf("TODO Отправляем данные в дискорд гильдии %v\n", killsDetails)

		store.UpdateBossKillsStorage(newKills)

		if len(kills.Data) > 0 {
			lastID = kills.Data[0].KillID
		}

		time.Sleep(2 * time.Minute)
	}
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
