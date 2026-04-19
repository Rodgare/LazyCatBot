package storage

import (
	"LazyCatBot/internal/sirus"
	"database/sql"
	"fmt"

	_ "modernc.org/sqlite"
)

type LeaderboardStorage struct {
	db *sql.DB
}

func NewLeaderboardStorage(db *sql.DB) *LeaderboardStorage {
	return &LeaderboardStorage{db: db}
}

func (s *LeaderboardStorage) InitDB() error {
	query := `
    CREATE TABLE IF NOT EXISTS leaderboard (
        raid_id INTEGER,
        boss_id INTEGER,
        class_id INTEGER,
        spec_id INTEGER,
        player_name TEXT,
        rank INTEGER,
        dps INTEGER,
        PRIMARY KEY (raid_id, boss_id, class_id, spec_id, rank, player_name)
    );
    CREATE INDEX IF NOT EXISTS idx_rank_lookup ON leaderboard (raid_id, boss_id, class_id, spec_id, dps DESC);
    `

	_, err := s.db.Exec(query)
	return err
}

func (s *LeaderboardStorage) UpdateLeaderboardStorage(raidOrder, encounter int, players []sirus.LeaderboardPlayer) error {
	tx, err := s.db.Begin()
	if err != nil {
		return err
	}
	defer tx.Rollback()

	_, err = tx.Exec("DELETE FROM leaderboard WHERE raid_id=? AND boss_id=?",
		raidOrder, encounter)
	if err != nil {
		return err
	}

	stmt, err := tx.Prepare(`INSERT OR REPLACE INTO leaderboard (raid_id, boss_id, class_id, spec_id, player_name, rank, dps) 
		VALUES (?, ?, ?, ?, ?, ?, ?)`)
	if err != nil {
		return err
	}
	defer stmt.Close()

	for _, p := range players {
		_, err = stmt.Exec(raidOrder, encounter, p.ClassID, p.SpecID, p.Name, p.Rank, p.Dps)

		if err != nil {
			return err
		}
	}

	err = tx.Commit()
	if err == nil {
		fmt.Printf("[DB] Успешно сохранено %d игроков для босса %d (рейд %d)\n", len(players), encounter, raidOrder)
	}
	return err
}

func (s *LeaderboardStorage) GetSpecRank(raid, boss, class, spec, dps int) int {
	var rank int

	err := s.db.QueryRow(`
		SELECT COUNT(*) + 1
		FROM leaderboard
		WHERE raid_id=? AND boss_id=? AND class_id=? AND spec_id=? AND dps > ?`,
		raid, boss, class, spec, dps).Scan(&rank)
	if err != nil {
		return 0
	}

	return rank
}
