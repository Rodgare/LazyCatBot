package worker

import (
	"LazyCatBot/internal/sirus"
	"LazyCatBot/internal/storage"
	"fmt"
	"log"
	"time"
)

func StartLeaderboardSync(store *storage.LeaderboardStorage) {
	fmt.Println("[Worker] Starting sync system...")

	for {
		fmt.Println("[Worker] Starting data update cycle...")

		actualRaids, err := sirus.FetchActualRaids()
		if err != nil {
			log.Printf("[Worker] Error fetching actual raids: %v", err)
			time.Sleep(1 * time.Minute)
			continue
		}

		for _, raid := range actualRaids {
			// if !raid.Actual {
			// 	continue
			// }

			fmt.Printf("[Worker] === Processing actual raid: %s (ID: %d) ===\n", raid.MapName, raid.Order)

			for bossID, encounter := range raid.Encounters {
				fmt.Printf("[Worker] Parsing boss %s (R:%d, B:%d)...\n", encounter.Name, raid.Order, bossID)

				players, err := sirus.FetchFullLeaderboard(raid.Order, bossID)
				if err != nil {
					log.Printf("[Worker] Error (Raid:%d Boss:%d): %v", raid.Order, bossID, err)
					time.Sleep(1 * time.Hour)
					continue
				}

				if len(players) == 0 {
					fmt.Printf("[Worker] No player data for Raid:%d Boss:%d\n", raid.Order, bossID)
					time.Sleep(10 * time.Second)
					continue
				}

				err = store.UpdateLeaderboardStorage(raid.Order, bossID, players)
				if err != nil {
					log.Printf("[Worker] Error saving to database: %v\n", err)
					time.Sleep(1 * time.Hour)
				}

				time.Sleep(5 * time.Second)
			}
		}

		fmt.Println("[Worker] All actual data updated. Sleeping 12 hours...")
		time.Sleep(12 * time.Hour)
	}
}
