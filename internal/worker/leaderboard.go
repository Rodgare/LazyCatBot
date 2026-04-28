package worker

import (
	"LazyCatBot/internal/sirus"
	"LazyCatBot/internal/storage"
	"fmt"
	"log"
	"time"

	"github.com/robfig/cron/v3"
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
				classes := sirus.GetSpecs()

				for classID, specs := range classes {
					for specID := range specs {
						for ilvlFrom, ilvlTo := 0, 5; ilvlTo < 310; ilvlFrom, ilvlTo = ilvlFrom+5, ilvlTo+5 {
							fmt.Printf("[Worker] Parsing boss %s (R:%d, B:%d)...\n", encounter.Name, raid.Order, bossID)

							players, err := sirus.FetchLeaderboard(raid.Order, bossID, classID, specID, ilvlFrom, ilvlTo)
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

							err = store.UpdateLeaderboardStorage(raid.Order, bossID, classID, specID, ilvlFrom, ilvlTo, players)
							if err != nil {
								log.Printf("[Worker] Error saving to database: %v\n", err)
								time.Sleep(1 * time.Hour)
							}

							time.Sleep(5 * time.Second)
						}

					}
				}

			}
		}

		fmt.Println("[Worker] All actual data updated. Sleeping 12 hours...")
		time.Sleep(12 * time.Hour)
	}
}

func StartCronScheduler(lbStore *storage.LeaderboardStorage) {
	msk := time.FixedZone("MSK", 3*3600)

	c := cron.New(cron.WithLocation(msk))

	_, err := c.AddFunc("0 3 * * *", func() {
		log.Println("[Cron] 03:00 MSK: Погнали синхронизировать Sirus...")
		StartLeaderboardSync(lbStore)
	})

	if err != nil {
		log.Printf("[Cron] Критическая ошибка планировщика: %v", err)
		return
	}

	c.Start()
	log.Println("[Cron] Планировщик успешно запущен на 03:00 MSK")
}
