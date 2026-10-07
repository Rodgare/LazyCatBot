package worker

import (
	"time"
)

// StartMythicLeaderboardSync downloads the challenge scores leaderboard for each
// realm and stores it in the mythic_leaderboard table. Used for coloring Rio in
// mythic reports by spec percentile.
func (w *Worker) StartMythicLeaderboardSync() {
	if w.mythLbStore == nil {
		w.logger.Warn("Mythic leaderboard store is not wired, skipping sync")
		return
	}

	season := w.cfg.MythicSeason
	weekID := w.cfg.MythicWeekID

	realms := []string{"x3", "x5"}
	for _, realm := range realms {
		w.logger.Info("Start parsing mythic challenge scores leaderboard", "realm", realm, "season", season, "week_id", weekID)

		players, err := w.sirusClient.FetchChallengeScores(realm, season, weekID)
		if err != nil {
			w.logger.Error("Fetch challenge scores error", "error", err, "realm", realm)
			continue
		}

		if len(players) == 0 {
			w.logger.Error("No mythic leaderboard data", "realm", realm)
			continue
		}

		if err := w.mythLbStore.UpdateMythicLeaderboard(realm, season, weekID, players); err != nil {
			w.logger.Error("Error saving mythic leaderboard", "error", err, "realm", realm)
			continue
		}

		time.Sleep(5 * time.Second)
	}

	w.logger.Info("Mythic leaderboard sync finished")
}