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
        ilvl INTEGER,
        dps INTEGER,
        PRIMARY KEY (raid_id, boss_id, class_id, spec_id, player_name)
    );
    CREATE INDEX IF NOT EXISTS idx_rank_lookup ON leaderboard (raid_id, boss_id, class_id, spec_id, dps DESC);
    `

	_, err := s.db.Exec(query)
	return err
}

func (s *LeaderboardStorage) UpdateLeaderboardStorage(raidOrder, encounter int, players []sirus.LeaderboardPlayer) error {
	if len(players) < 10 {
		return fmt.Errorf("Less then 10 players for Raid: %d, Endounter: %d", raidOrder, encounter)
	}
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

	stmt, err := tx.Prepare(`INSERT OR REPLACE INTO leaderboard (raid_id, boss_id, class_id, spec_id, player_name, ilvl, dps) 
		VALUES (?, ?, ?, ?, ?, ?, ?)`)
	if err != nil {
		return err
	}
	defer stmt.Close()

	for _, p := range players {
		_, err = stmt.Exec(raidOrder, encounter, p.ClassID, p.SpecID, p.Name, p.Ilvl, p.Dps)

		if err != nil {
			return err
		}
	}

	err = tx.Commit()
	if err == nil {
		fmt.Printf("[DB] Successfully saved %d players for boss %d (raid %d)\n", len(players), encounter, raidOrder)
	}
	return err
}

func (s *LeaderboardStorage) GetRank(raid, boss int, p sirus.Player) (int, int, int, int, int, int, int, int, error) {
	var ilvlRank, ilvlTotal, specRank, specTotal, classRank, classTotal, overallRank, overallTotal int
	var specPrcnt, classPrcnt, ilvlPrcnt, overallPrcnt int

	minIlvl := (p.Ilvl / 5) * 5
	maxIlvl := minIlvl + 4

	query := `SELECT 
	(SELECT COUNT(*) + 1 FROM leaderboard WHERE raid_id = ? AND boss_id = ? AND spec_id = ? AND ilvl BETWEEN ? AND ? AND dps > ?) as ilvl_rank,
	(SELECT COUNT(*) FROM leaderboard WHERE raid_id = ? AND boss_id = ? AND spec_id = ? AND ilvl BETWEEN ? AND ?) as ilvl_total,
    (SELECT COUNT(*) + 1 FROM leaderboard WHERE raid_id = ? AND boss_id = ? AND spec_id = ? AND dps > ?) as spec_rank,
    (SELECT COUNT(*) FROM leaderboard WHERE raid_id = ? AND boss_id = ? AND spec_id = ?) as spec_total,
    (SELECT COUNT(*) + 1 FROM leaderboard WHERE raid_id = ? AND boss_id = ? AND class_id = ? AND dps > ?) as class_rank,
    (SELECT COUNT(*) FROM leaderboard WHERE raid_id = ? AND boss_id = ? AND class_id = ?) as class_total,
    (SELECT COUNT(*) + 1 FROM leaderboard WHERE raid_id = ? AND boss_id = ? AND dps > ?) as overall_rank,
    (SELECT COUNT(*) FROM leaderboard WHERE raid_id = ? AND boss_id = ?) as overall_total;`

	err := s.db.QueryRow(query,
		raid, boss, p.Spec, minIlvl, maxIlvl, p.Dps, // ilvl_rank
		raid, boss, p.Spec, minIlvl, maxIlvl,        // ilvl_total
		raid, boss, p.Spec, p.Dps,                   // spec_rank
		raid, boss, p.Spec,                          // spec_total
		raid, boss, p.ClassID, p.Dps,                // class_rank
		raid, boss, p.ClassID,                       // class_total
		raid, boss, p.Dps,                           // overall_rank
		raid, boss,                                  // overall_total
	).Scan(&ilvlRank, &ilvlTotal, &specRank, &specTotal, &classRank, &classTotal, &overallRank, &overallTotal)

	if err != nil {
		return 0, 0, 0, 0, 0, 0, 0, 0, err
	}

	if ilvlTotal > 0 {
		ilvlPrcnt = int(float64(ilvlTotal-ilvlRank+1) / float64(ilvlTotal) * 100)
	}
	if specTotal > 0 {
		specPrcnt = int(float64(specTotal-specRank+1) / float64(specTotal) * 100)
	}
	if classTotal > 0 {
		classPrcnt = int(float64(classTotal-classRank+1) / float64(classTotal) * 100)
	}
	if overallTotal > 0 {
		overallPrcnt = int(float64(overallTotal-overallRank+1) / float64(overallTotal) * 100)
	}

	return specRank, specPrcnt, classRank, classPrcnt, ilvlRank, ilvlPrcnt, overallRank, overallPrcnt, nil
}
