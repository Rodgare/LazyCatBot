package worker

import (
	"LazyCatBot/internal/sirus"
	"LazyCatBot/internal/storage"
	"fmt"
	"log"
	"time"
)

func StartMetasirusLbSync(store *storage.LeaderboardStorage) {
	fmt.Println("[Worker] Starting metasirus sync system...")

	for {
		raids, err := sirus.FetchActualRaids()
		if err != nil {
			log.Printf("[Worker] Error fetching actual raids: %v", err)
			time.Sleep(1 * time.Minute)
			continue
		}

		for _, raid := range raids {
			fmt.Printf("[Worker] === Processing raid: %s (ID: %d) ===\n", raid.MapName, raid.Order)

			for sirusBossID := range raid.Encounters {
				metasirusBossID := sirus.GetMetasirusBossID(raid.Order, sirusBossID)
				if metasirusBossID == 0 {
					continue
				}

				players, err := sirus.FetchMetasirusLeaderboard(raid.MapID, metasirusBossID, raid.Difficulty)
				if err != nil {
					log.Printf("[Worker] Error (Raid:%d Boss:%d): %v", raid.Order, sirusBossID, err)
					time.Sleep(1 * time.Hour)
					continue
				}

				if len(players) == 0 {
					fmt.Printf("[Worker] No player data for Raid:%d SirusBossID:%d metasirusBossID%d, \n", raid.Order, sirusBossID, metasirusBossID)
					time.Sleep(10 * time.Second)
					continue
				}

				err = store.UpdateLeaderboardStorage(raid.Order, sirusBossID, players)
				if err != nil {
					log.Printf("[Worker] Error saving to database: %v\n", err)
					time.Sleep(1 * time.Hour)
				}

				time.Sleep(5 * time.Second)
			}
		}

		fmt.Println("[Worker] All actual data updated. Sleeping 24 hours...")
		time.Sleep(24 * time.Hour)
	}
}
