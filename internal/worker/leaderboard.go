package worker

import (
	"LazyCatBot/internal/sirus"
	"time"

	"github.com/robfig/cron/v3"
)

func (w *Worker) StartLeaderboardSync() {
	w.logger.Info("Starting leaderboard sync system...")
	realms := []string{"x3", "x1", "x5"}

	for _, r := range realms {
		actualRaids, err := w.sirusClient.FetchActualRaids(r)
		if err != nil {
			w.logger.Error("Error fetching actual raids", "realm", r, "error", err)
			continue
		}

		w.arStore.ResetActualRaids(r)

		for _, raid := range actualRaids {
			for bossID, encounter := range raid.Encounters {
				if raid.Actual && !sirus.IsRaidBannedForDpsMeter(raid.Order) {
					w.arStore.UpdateActualRaids(raid.Order, bossID, r)
				}

				classes := sirus.GetSpecs()

				for classID, specs := range classes {
					for specID := range specs {
						l := w.logger.With(
							"realm", r,
							"raid_name", encounter.Name,
							"raid_id", raid.Order,
							"boss_id", bossID,
							"class_id", classID,
							"spec_id", specID,
						)

						l.Info("Start parsing boss leaderboard")

						role := sirus.GetRoleString(classID, specID)
						players, err := w.sirusClient.FetchLeaderboard(r, raid.Order, bossID, classID, specID, role)
						if err != nil {
							l.Error("Fetch leaderboard error", "error", err)
							time.Sleep(2 * time.Second)
							continue
						}

						if len(players) == 0 {
							l.Error("No player data")
							time.Sleep(1 * time.Second)
							continue
						}

						err = w.lbStore.UpdateLeaderboardStorage(r, raid.Order, bossID, classID, specID, players)
						if err != nil {
							l.Error("Error saving to database", "error", err)
							time.Sleep(5 * time.Second)
						}

						time.Sleep(5 * time.Second)

					}
				}

			}
		}
	}

	w.logger.Info("All actual data updated. Sleeping to next 3:00 a.m...")
}

func (w *Worker) StartCronScheduler() {
	msk := time.FixedZone("MSK", 3*3600)

	c := cron.New(cron.WithLocation(msk))

	_, err := c.AddFunc("0 3 * * *", func() {
		w.logger.Info("[Cron] 03:00: cron tast is started")
		w.StartLeaderboardSync()
	})

	if err != nil {
		w.logger.Error("Cron error", "error", err)
		return
	}

	c.Start()
	w.logger.Info("Cron task started successfully")
}
