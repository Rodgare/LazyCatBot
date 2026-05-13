package worker

import (
	"LazyCatBot/internal/sirus"
	"LazyCatBot/internal/storage"
	"fmt"
	"time"
)

func (w *Worker) StartMetasirusLbSync(store *storage.LeaderboardStorage) {
	fmt.Println("[Worker] Starting metasirus sync system...")

	for {
		raids, err := w.sirusClient.FetchActualRaids()
		if err != nil {
			w.logger.Error("Error fetching actual raids", "error", err)
			time.Sleep(1 * time.Minute)
			continue
		}

		for _, raid := range raids {
			w.logger.Info("Processing raid", "raid_name", raid.MapName, "raid_id", raid.Order)

			for sirusBossID := range raid.Encounters {
				metasirusBossID := sirus.GetMetasirusBossID(raid.Order, sirusBossID)
				if metasirusBossID == 0 {
					continue
				}

				players, err := w.sirusClient.FetchMetasirusLeaderboard(raid.MapID, metasirusBossID, raid.Difficulty)
				if err != nil {
					w.logger.Error("Get metasirus data error", "error", err, "raid_id", raid.Order, "boss_id", sirusBossID)
					time.Sleep(1 * time.Hour)
					continue
				}

				if len(players) == 0 {
					w.logger.Error("No player data", "raid_id", raid.Order, "boss_id", sirusBossID, "metasirus_boss_id", metasirusBossID)
					time.Sleep(10 * time.Second)
					continue
				}

				// err = store.UpdateLeaderboardStorage(raid.Order, sirusBossID, players)
				// if err != nil {
				// 	log.Printf("[Worker] Error saving to database: %v\n", err)
				// 	time.Sleep(1 * time.Hour)
				// }

				time.Sleep(5 * time.Second)
			}
		}

		w.logger.Info("All actual data updated. Sleeping 24 hours...")
		time.Sleep(24 * time.Hour)
	}
}
