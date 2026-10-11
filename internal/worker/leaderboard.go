package worker

import (
	"LazyCatBot/internal/sirus"
	"time"

	"github.com/robfig/cron/v3"
)

func (w *Worker) StartLeaderboardSync() {
	w.logger.Info("Starting leaderboard sync system...")
	for _, r := range w.cfg.Realms {
		w.syncRealm(r, false)
	}
	w.logger.Info("All actual data updated. Sleeping to next 3:00 a.m...")
}

// StartLeaderboardSyncOnStart runs the raid leaderboard sync once at startup for
// the realms selected by config (defaults to all configured realms). With
// force=true it ignores the per-day "already synced" gate so data can be
// re-parsed after code changes.
func (w *Worker) StartLeaderboardSyncOnStart() {
	realms := w.cfg.SyncRealms
	if len(realms) == 0 {
		realms = w.cfg.Realms
	}
	w.logger.Info("Startup leaderboard sync requested", "realms", realms, "force", w.cfg.SyncForce)
	for _, r := range realms {
		w.syncRealm(r, w.cfg.SyncForce)
	}
	w.logger.Info("Startup leaderboard sync finished")
}

func (w *Worker) syncRealm(r string, force bool) {
	syncDate := time.Now().In(time.FixedZone("MSK", 3*3600)).Format("2006-01-02")

	actualRaids, err := w.sirusClient.FetchActualRaids(r)
	if err != nil {
		w.logger.Error("Error fetching actual raids", "realm", r, "error", err)
		return
	}

	onlyActual := w.cfg.IsSecondaryRealm(r)

	w.arStore.ResetActualRaids(r)

	for _, raid := range actualRaids {
		if onlyActual && !raid.Actual {
			w.logger.Info("Skipping non-actual raid on secondary realm",
				"realm", r, "raid_id", raid.Order, "raid_name", raid.MapName)
			continue
		}

		if raid.Actual && !sirus.IsRaidBannedForDpsMeter(raid.Order) {
			for bossID := range raid.Encounters {
				w.arStore.UpdateActualRaids(raid.Order, bossID, r)
			}
		}

		for bossID, encounter := range raid.Encounters {
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

					if !force {
						done, err := w.lbStore.IsLeaderboardSyncDone(r, syncDate, raid.Order, bossID, classID, specID)
						if err != nil {
							l.Error("Check leaderboard sync state err", "error", err)
							continue
						}
						if done {
							continue
						}
					}

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
						continue
					}

					err = w.lbStore.MarkLeaderboardSyncDone(r, syncDate, raid.Order, bossID, classID, specID)
					if err != nil {
						l.Error("Mark leaderboard sync state err", "error", err)
					}

					time.Sleep(5 * time.Second)

				}
			}

		}
	}
}

// RequestLeaderboardSync starts a leaderboard resync in the background, targeting
// a single realm (empty = all configured realms). With force=true it re-parses
// even combos already marked synced for today. Admin command entry point.
func (w *Worker) RequestLeaderboardSync(realm string, force bool) {
	go func() {
		w.resyncMu.Lock()
		defer w.resyncMu.Unlock()
		realms := w.cfg.Realms
		if realm != "" {
			realms = []string{realm}
		}
		w.logger.Info("Leaderboard resync via admin command", "realms", realms, "force", force)
		for _, r := range realms {
			w.syncRealm(r, force)
		}
		w.logger.Info("Leaderboard resync finished")
	}()
}

func (w *Worker) StartCronScheduler() {
	msk := time.FixedZone("MSK", 3*3600)

	c := cron.New(cron.WithLocation(msk))

	_, err := c.AddFunc("0 3 * * *", func() {
		w.logger.Info("[Cron] 03:00: raid leaderboard sync started")
		w.StartLeaderboardSync()
	})

	if err != nil {
		w.logger.Error("Cron error", "error", err)
		return
	}

	_, err = c.AddFunc("0 18 * * *", func() {
		w.logger.Info("[Cron] 18:00: mythic leaderboard sync started")
		w.StartMythicLeaderboardSync()
	})

	if err != nil {
		w.logger.Error("Cron error", "error", err)
		return
	}

	c.Start()
	w.logger.Info("Cron task started successfully")
}
