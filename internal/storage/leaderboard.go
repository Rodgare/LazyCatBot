package storage

import (
	"LazyCatBot/internal/models"
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
		guild_id INTEGER,
		zodiac INTEGER,
		t4 INTEGER,
        dps INTEGER,
		hps INTEGER,
        PRIMARY KEY (raid_id, boss_id, class_id, spec_id, player_name)
    );
    CREATE INDEX IF NOT EXISTS idx_rank_lookup ON leaderboard (raid_id, boss_id, class_id, spec_id, dps DESC);
	CREATE INDEX IF NOT EXISTS idx_hps_rank_lookup ON leaderboard (raid_id, boss_id, class_id, spec_id, hps DESC);
    `

	_, err := s.db.Exec(query)
	return err
}

func (s *LeaderboardStorage) UpdateLeaderboardStorage(raidOrder, encounter, classID, specID int, players []models.LeaderboardPlayer) error {
	if len(players) < 2 {
		return fmt.Errorf("Less then 2 players for Raid: %d, Encounter: %d", raidOrder, encounter)
	}
	tx, err := s.db.Begin()
	if err != nil {
		return err
	}
	defer tx.Rollback()

	_, err = tx.Exec("DELETE FROM leaderboard WHERE raid_id=? AND boss_id=? AND class_id=? AND spec_id=?",
		raidOrder, encounter, classID, specID)
	if err != nil {
		return err
	}

	stmt, err := tx.Prepare(`INSERT OR REPLACE INTO leaderboard (raid_id, boss_id, class_id, spec_id, player_name, ilvl, guild_id, zodiac, t4, dps, hps) 
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`)
	if err != nil {
		return err
	}
	defer stmt.Close()

	for _, p := range players {
		t4 := sirus.GetT4CountForLeaderboard(p.Itemset)
		_, err = stmt.Exec(raidOrder, encounter, p.ClassID, p.SpecID, p.Name, p.Ilvl, p.GuildID, p.Zodiac.ID, t4, p.Dps, p.Hps)

		if err != nil {
			return err
		}
	}

	err = tx.Commit()
	if err == nil {
		fmt.Printf("[DB] Successfully saved %d players R: %d B: %d classID: %d, specID: %d\n", len(players), raidOrder, encounter, classID, specID)
	}
	return err
}

func (s *LeaderboardStorage) UpsertPlayer(raid, boss int, p models.Player, role string) error {
	condition := "excluded.dps > leaderboard.dps"
	if role == "heal" {
		condition = "excluded.hps > leaderboard.hps"
	}
	t4 := sirus.GetT4Count(p.Itemset)

	query := fmt.Sprintf(`
		INSERT INTO leaderboard (raid_id, boss_id, class_id, spec_id, player_name, ilvl, guild_id, zodiac, t4, dps, hps) 
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
		ON CONFLICT(raid_id, boss_id, class_id, spec_id, player_name) 
		DO UPDATE SET 
    		ilvl = excluded.ilvl,
			zodiac = excluded.zodiac,
			t4 = excluded.t4,
    		dps = excluded.dps,
    		hps = excluded.hps
		WHERE %s;
	`, condition)
	_, err := s.db.Exec(query, raid, boss, p.ClassID, p.Spec, p.Name, p.Ilvl, p.Guild.ID, p.Zodiac.ID, t4, p.Dps, p.Hps)
	return err
}

func (s *LeaderboardStorage) GetPlayerRank(raid, boss int, p models.PlayerReport, role string) (models.PlayerReport, error) {
	var ilvlRank, ilvlTotal, specRank, specTotal, classRank, classTotal, overallRank, overallTotal int

	valField := "dps"
	if role == "hps" {
		valField = "hps"
	}

	query := fmt.Sprintf(`SELECT 
		(SELECT COUNT(*) + 1 FROM leaderboard WHERE raid_id = ? AND boss_id = ? AND class_id = ? AND spec_id = ? AND ilvl <= ? AND %s > ?) as ilvl_rank,
		(SELECT COUNT(*) FROM leaderboard WHERE raid_id = ? AND boss_id = ? AND class_id = ? AND spec_id = ? AND ilvl <= ?) as ilvl_total,
		(SELECT COUNT(*) + 1 FROM leaderboard WHERE raid_id = ? AND boss_id = ? AND class_id = ? AND spec_id = ? AND %s > ?) as spec_rank,
		(SELECT COUNT(*) FROM leaderboard WHERE raid_id = ? AND boss_id = ? AND class_id = ? AND spec_id = ?) as spec_total,
		(SELECT COUNT(*) + 1 FROM leaderboard WHERE raid_id = ? AND boss_id = ? AND class_id = ? AND %s > ?) as class_rank,
		(SELECT COUNT(*) FROM leaderboard WHERE raid_id = ? AND boss_id = ? AND class_id = ?) as class_total,
		(SELECT COUNT(*) + 1 FROM leaderboard WHERE raid_id = ? AND boss_id = ? AND %s > ?) as overall_rank,
		(SELECT COUNT(*) FROM leaderboard WHERE raid_id = ? AND boss_id = ? AND %s > 0) as overall_total;`,
		valField, valField, valField, valField, valField)

	val := p.Dps
	if role == "hps" {
		val = p.Hps
	}
	err := s.db.QueryRow(query,
		raid, boss, p.ClassID, p.SpecID, p.Ilvl, val, // ilvl_rank
		raid, boss, p.ClassID, p.SpecID, p.Ilvl, // ilvl_total
		raid, boss, p.ClassID, p.SpecID, val, // spec_rank
		raid, boss, p.ClassID, p.SpecID, // spec_total
		raid, boss, p.ClassID, val, // class_rank
		raid, boss, p.ClassID, // class_total
		raid, boss, val, // overall_rank
		raid, boss, // overall_total
	).Scan(&ilvlRank, &ilvlTotal, &specRank, &specTotal, &classRank, &classTotal, &overallRank, &overallTotal)
	if err != nil {
		return p, err
	}

	p.SpecRank = specRank
	p.ClassRank = classRank
	p.IlvlRank = ilvlRank
	p.OverallRank = overallRank

	if specTotal > 0 {
		p.SpecPercentile = int(float64(specTotal-specRank+1) / float64(specTotal) * 100)
	}
	if classTotal > 0 {
		p.ClassPercentile = int(float64(classTotal-classRank+1) / float64(classTotal) * 100)
	}
	if ilvlTotal > 0 {
		p.IlvlPercentile = int(float64(ilvlTotal-ilvlRank+1) / float64(ilvlTotal) * 100)
	}
	if overallTotal > 0 {
		p.OverallPercentile = int(float64(overallTotal-overallRank+1) / float64(overallTotal) * 100)
	}
	return p, nil
}

// func (s *LeaderboardStorage) GetDpsRank(raid, boss int, p models.Player) (int, int, int, int, int, int, int, int, error) {
// 	var ilvlRank, ilvlTotal, specRank, specTotal, classRank, classTotal, overallRank, overallTotal int
// 	var specPrcnt, classPrcnt, ilvlPrcnt, overallPrcnt int

// 	// minIlvl := (p.Ilvl / 5) * 5
// 	// maxIlvl := minIlvl + 4

// 	query := `SELECT
// 	(SELECT COUNT(*) + 1 FROM leaderboard WHERE raid_id = ? AND boss_id = ? AND class_id = ? AND spec_id = ? AND ilvl <= ? AND dps > ?) as ilvl_rank,
// 	(SELECT COUNT(*) FROM leaderboard WHERE raid_id = ? AND boss_id = ? AND class_id = ? AND spec_id = ? AND ilvl <= ?) as ilvl_total,
//     (SELECT COUNT(*) + 1 FROM leaderboard WHERE raid_id = ? AND boss_id = ? AND class_id = ? AND spec_id = ? AND dps > ?) as spec_rank,
//     (SELECT COUNT(*) FROM leaderboard WHERE raid_id = ? AND boss_id = ? AND class_id = ? AND spec_id = ?) as spec_total,
//     (SELECT COUNT(*) + 1 FROM leaderboard WHERE raid_id = ? AND boss_id = ? AND class_id = ? AND dps > ?) as class_rank,
//     (SELECT COUNT(*) FROM leaderboard WHERE raid_id = ? AND boss_id = ? AND class_id = ?) as class_total,
//     (SELECT COUNT(*) + 1 FROM leaderboard WHERE raid_id = ? AND boss_id = ? AND dps > ?) as overall_rank,
//     (SELECT COUNT(*) FROM leaderboard WHERE raid_id = ? AND boss_id = ? AND dps > 0) as overall_total;`

// 	err := s.db.QueryRow(query,
// 		raid, boss, p.ClassID, p.Spec, p.Ilvl, p.Dps, // ilvl_rank
// 		raid, boss, p.ClassID, p.Spec, p.Ilvl, // ilvl_total
// 		raid, boss, p.ClassID, p.Spec, p.Dps, // spec_rank
// 		raid, boss, p.ClassID, p.Spec, // spec_total
// 		raid, boss, p.ClassID, p.Dps, // class_rank
// 		raid, boss, p.ClassID, // class_total
// 		raid, boss, p.Dps, // overall_rank
// 		raid, boss, // overall_total
// 	).Scan(&ilvlRank, &ilvlTotal, &specRank, &specTotal, &classRank, &classTotal, &overallRank, &overallTotal)

// 	if err != nil {
// 		return 0, 0, 0, 0, 0, 0, 0, 0, err
// 	}

// 	if ilvlTotal > 0 {
// 		ilvlPrcnt = int(float64(ilvlTotal-ilvlRank+1) / float64(ilvlTotal) * 100)
// 	}
// 	if specTotal > 0 {
// 		specPrcnt = int(float64(specTotal-specRank+1) / float64(specTotal) * 100)
// 	}
// 	if classTotal > 0 {
// 		classPrcnt = int(float64(classTotal-classRank+1) / float64(classTotal) * 100)
// 	}
// 	if overallTotal > 0 {
// 		overallPrcnt = int(float64(overallTotal-overallRank+1) / float64(overallTotal) * 100)
// 	}

// 	return specRank, specPrcnt, classRank, classPrcnt, ilvlRank, ilvlPrcnt, overallRank, overallPrcnt, nil
// }

// func (s *LeaderboardStorage) GetHpsRank(raid, boss int, p models.Player) (int, int, int, int, int, int, int, int, error) {
// 	var ilvlRank, ilvlTotal, specRank, specTotal, classRank, classTotal, overallRank, overallTotal int
// 	var specPrcnt, classPrcnt, ilvlPrcnt, overallPrcnt int

// 	// minIlvl := (p.Ilvl / 5) * 5
// 	// maxIlvl := minIlvl + 4

// 	query := `SELECT
// 	(SELECT COUNT(*) + 1 FROM leaderboard WHERE raid_id = ? AND boss_id = ? AND class_id = ? AND spec_id = ? AND ilvl <= ? AND hps > ?) as ilvl_rank,
// 	(SELECT COUNT(*) FROM leaderboard WHERE raid_id = ? AND boss_id = ? AND class_id = ? AND spec_id = ? AND ilvl <= ?) as ilvl_total,
//     (SELECT COUNT(*) + 1 FROM leaderboard WHERE raid_id = ? AND boss_id = ? AND class_id = ? AND spec_id = ? AND hps > ?) as spec_rank,
//     (SELECT COUNT(*) FROM leaderboard WHERE raid_id = ? AND boss_id = ? AND class_id = ? AND spec_id = ?) as spec_total,
//     (SELECT COUNT(*) + 1 FROM leaderboard WHERE raid_id = ? AND boss_id = ? AND class_id = ? AND hps > ?) as class_rank,
//     (SELECT COUNT(*) FROM leaderboard WHERE raid_id = ? AND boss_id = ? AND class_id = ?) as class_total,
//     (SELECT COUNT(*) + 1 FROM leaderboard WHERE raid_id = ? AND boss_id = ? AND hps > ?) as overall_rank,
//     (SELECT COUNT(*) FROM leaderboard WHERE raid_id = ? AND boss_id = ? AND hps > 0) as overall_total;`

// 	err := s.db.QueryRow(query,
// 		raid, boss, p.ClassID, p.Spec, p.Ilvl, p.Hps, // ilvl_rank
// 		raid, boss, p.ClassID, p.Spec, p.Ilvl, // ilvl_total
// 		raid, boss, p.ClassID, p.Spec, p.Hps, // spec_rank
// 		raid, boss, p.ClassID, p.Spec, // spec_total
// 		raid, boss, p.ClassID, p.Hps, // class_rank
// 		raid, boss, p.ClassID, // class_total
// 		raid, boss, p.Hps, // overall_rank
// 		raid, boss, // overall_total
// 	).Scan(&ilvlRank, &ilvlTotal, &specRank, &specTotal, &classRank, &classTotal, &overallRank, &overallTotal)

// 	if err != nil {
// 		return 0, 0, 0, 0, 0, 0, 0, 0, err
// 	}

// 	if ilvlTotal > 0 {
// 		ilvlPrcnt = int(float64(ilvlTotal-ilvlRank+1) / float64(ilvlTotal) * 100)
// 	}
// 	if specTotal > 0 {
// 		specPrcnt = int(float64(specTotal-specRank+1) / float64(specTotal) * 100)
// 	}
// 	if classTotal > 0 {
// 		classPrcnt = int(float64(classTotal-classRank+1) / float64(classTotal) * 100)
// 	}
// 	if overallTotal > 0 {
// 		overallPrcnt = int(float64(overallTotal-overallRank+1) / float64(overallTotal) * 100)
// 	}

// 	return specRank, specPrcnt, classRank, classPrcnt, ilvlRank, ilvlPrcnt, overallRank, overallPrcnt, nil
// }

func (s *LeaderboardStorage) GetBossTop(raidID, bossID, guildID int, role string) ([]models.PlayerReport, error) {
	orderBy := "dps"
	if role == "hps" {
		orderBy = "hps"
	}

	query := fmt.Sprintf(`
		SELECT player_name, class_id, spec_id, ilvl, zodiac, t4, dps, hps
		FROM leaderboard
		WHERE raid_id = ? AND boss_id = ? AND guild_id = ?
		ORDER BY %s DESC
		LIMIT 30`, orderBy)

	rows, err := s.db.Query(query, raidID, bossID, guildID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var players []models.PlayerReport
	for rows.Next() {
		var p models.PlayerReport
		err := rows.Scan(&p.Name, &p.ClassID, &p.SpecID, &p.Ilvl, &p.Zodiac, &p.T4, &p.Dps, &p.Hps)
		if err != nil {
			return nil, err
		}
		p.Role = sirus.GetRole(p.ClassID, p.SpecID)
		p.SpecName = sirus.GetSpecName(p.ClassID, p.SpecID)
		players = append(players, p)
	}
	return players, nil
}
