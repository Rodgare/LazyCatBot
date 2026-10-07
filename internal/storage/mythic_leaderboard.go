package storage

import (
	"LazyCatBot/internal/models"
	"database/sql"
	"log/slog"

	_ "modernc.org/sqlite"
)

type MythicLeaderboardStorage struct {
	db     *sql.DB
	logger *slog.Logger
}

func NewMythicLeaderboardStorage(db *sql.DB, logger *slog.Logger) *MythicLeaderboardStorage {
	return &MythicLeaderboardStorage{db: db, logger: logger}
}

// UpdateMythicLeaderboard replaces the whole mythic leaderboard snapshot for the
// given realm/season/week in one transaction.
func (s *MythicLeaderboardStorage) UpdateMythicLeaderboard(realm string, season, weekID int, players []models.MythicScorePlayer) error {
	if realm == "" {
		realm = "x3"
	}

	tx, err := s.db.Begin()
	if err != nil {
		return err
	}
	defer tx.Rollback()

	if _, err := tx.Exec("DELETE FROM mythic_leaderboard WHERE realm=? AND season=? AND week_id=?", realm, season, weekID); err != nil {
		return err
	}

	stmt, err := tx.Prepare(`INSERT OR REPLACE INTO mythic_leaderboard
		(realm, guid, name, class_id, spec_id, current_score, position, total_runs, timed_runs, best_key, zodiac, season, week_id, updated_at)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, unixepoch())`)
	if err != nil {
		return err
	}
	defer stmt.Close()

	for _, p := range players {
		_, err = stmt.Exec(realm, p.GUID, p.Name, p.Class, p.SpecID, p.Score, p.Position, p.TotalRuns, p.TimedRuns, p.BestKey, p.Zodiac, season, weekID)
		if err != nil {
			return err
		}
	}

	if err := tx.Commit(); err != nil {
		return err
	}
	s.logger.Info("[DB] Mythic leaderboard saved", "realm", realm, "season", season, "week_id", weekID, "players", len(players))
	return nil
}

// GetPlayerMythicRank computes the player's rank and percentile among the same
// class+spec by current_score, mirroring the raid GetPlayerRank logic.
func (s *MythicLeaderboardStorage) GetPlayerMythicRank(realm string, season, weekID, classID, specID int, score float64) (rank, total, percentile int, err error) {
	if realm == "" {
		realm = "x3"
	}

	err = s.db.QueryRow(`SELECT
		(SELECT COUNT(*) + 1 FROM mythic_leaderboard WHERE realm=? AND season=? AND week_id=? AND class_id=? AND spec_id=? AND current_score > ?) as rank,
		(SELECT COUNT(*) FROM mythic_leaderboard WHERE realm=? AND season=? AND week_id=? AND class_id=? AND spec_id=?) as total`,
		realm, season, weekID, classID, specID, score,
		realm, season, weekID, classID, specID,
	).Scan(&rank, &total)
	if err != nil {
		return 0, 0, 0, err
	}

	if total > 0 {
		percentile = int(float64(total-rank+1) / float64(total) * 100)
		if percentile > 100 {
			percentile = 100
		}
	}
	return rank, total, percentile, nil
}