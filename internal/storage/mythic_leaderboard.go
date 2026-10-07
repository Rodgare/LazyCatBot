package storage

import (
	"LazyCatBot/internal/models"
	"database/sql"
	"log/slog"
	"strings"

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

// GetGuildMythicPlayers returns the latest mythic leaderboard snapshot rows for
// the given guild member GUIDs, joined with guild ilvl and cached character
// data (black diamonds, title), sorted by current_score descending.
func (s *MythicLeaderboardStorage) GetGuildMythicPlayers(realm string, guids []int) ([]models.GuildMythicPlayer, error) {
	if realm == "" {
		realm = "x3"
	}
	if len(guids) == 0 {
		return nil, nil
	}

	placeholders := strings.TrimSuffix(strings.Repeat("?,", len(guids)), ",")
	args := make([]any, 0, len(guids)+2)
	args = append(args, realm, realm)
	for _, g := range guids {
		args = append(args, g)
	}

	query := `SELECT ml.guid, ml.name, ml.class_id, ml.spec_id, ml.current_score, ml.position,
	       ml.best_key, ml.zodiac, ml.total_runs, ml.timed_runs,
	       gm.ilvl, COALESCE(cc.black_diamonds, 0), COALESCE(cc.title, ''),
	       (SELECT COUNT(*)+1 FROM mythic_leaderboard ml2
	          WHERE ml2.realm = ml.realm AND ml2.class_id = ml.class_id
	            AND ml2.current_score > ml.current_score
	            AND ml2.updated_at = ml.updated_at) AS class_rank,
	       (SELECT COUNT(*) FROM mythic_leaderboard ml2
	          WHERE ml2.realm = ml.realm AND ml2.class_id = ml.class_id
	            AND ml2.updated_at = ml.updated_at) AS class_total,
	       (SELECT COUNT(*)+1 FROM mythic_leaderboard ml2
	          WHERE ml2.realm = ml.realm AND ml2.class_id = ml.class_id AND ml2.spec_id = ml.spec_id
	            AND ml2.current_score > ml.current_score
	            AND ml2.updated_at = ml.updated_at) AS spec_rank,
	       (SELECT COUNT(*) FROM mythic_leaderboard ml2
	          WHERE ml2.realm = ml.realm AND ml2.class_id = ml.class_id AND ml2.spec_id = ml.spec_id
	            AND ml2.updated_at = ml.updated_at) AS spec_total
	FROM mythic_leaderboard ml
	JOIN guild_members gm ON gm.id = ml.guid AND gm.realm = ml.realm
	LEFT JOIN character_cache cc ON cc.realm = ml.realm AND cc.name = ml.name
	WHERE ml.realm = ? AND ml.updated_at = (SELECT MAX(updated_at) FROM mythic_leaderboard WHERE realm = ?)
	  AND ml.guid IN (` + placeholders + `)
	ORDER BY ml.current_score DESC`

	rows, err := s.db.Query(query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var players []models.GuildMythicPlayer
	for rows.Next() {
		var p models.GuildMythicPlayer
		if err := rows.Scan(&p.GUID, &p.Name, &p.ClassID, &p.SpecID, &p.Score, &p.Position,
			&p.BestKey, &p.Zodiac, &p.TotalRuns, &p.TimedRuns,
			&p.Ilvl, &p.BlackDiamonds, &p.Title,
			&p.ClassRank, &p.ClassTotal, &p.SpecRank, &p.SpecTotal); err != nil {
			return nil, err
		}
		players = append(players, p)
	}
	return players, rows.Err()
}