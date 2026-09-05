package storage

import (
	"LazyCatBot/internal/models"
	"LazyCatBot/internal/sirus"
	"database/sql"
	"fmt"
	"log/slog"

	_ "modernc.org/sqlite"
)

type LeaderboardStorage struct {
	db     *sql.DB
	logger *slog.Logger
}

func NewLeaderboardStorage(db *sql.DB, logger *slog.Logger) *LeaderboardStorage {
	return &LeaderboardStorage{
		db:     db,
		logger: logger,
	}
}

func (s *LeaderboardStorage) UpdateLeaderboardStorage(realm string, raidOrder, encounter, classID, specID int, players []models.LeaderboardPlayer) error {
	if realm == "" {
		realm = "x3"
	}
	if len(players) < 2 {
		return fmt.Errorf("Less then 2 players for Raid: %d, Encounter: %d", raidOrder, encounter)
	}
	tx, err := s.db.Begin()
	if err != nil {
		return err
	}
	defer tx.Rollback()

	_, err = tx.Exec("DELETE FROM leaderboard WHERE raid_id=? AND boss_id=? AND class_id=? AND spec_id=? AND realm=?",
		raidOrder, encounter, classID, specID, realm)
	if err != nil {
		return err
	}

	stmt, err := tx.Prepare(`INSERT OR IGNORE INTO leaderboard (raid_id, boss_id, class_id, spec_id, player_name, ilvl, guild_id, zodiac, category, t4, role, dps, hps, realm) 
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`)
	if err != nil {
		return err
	}
	defer stmt.Close()

	role := sirus.GetRoleString(classID, specID)

	for _, p := range players {
		t4 := sirus.GetT4CountForLeaderboard(p.Itemset)
		_, err = stmt.Exec(raidOrder, encounter, p.ClassID, p.SpecID, p.Name, p.Ilvl, p.GuildID, p.Zodiac.ID, p.Category, t4, role, p.Dps, p.Hps, realm)

		if err != nil {
			return err
		}
	}

	err = tx.Commit()
	if err == nil {
		s.logger.Info("[DB] Successfully saved", "realm", realm, "players_count", len(players), "raid_id", raidOrder, "boss_id", encounter, "class_id", classID, "spec_id", specID)
	}
	return err
}

func (s *LeaderboardStorage) UpsertPlayer(realm string, raid, boss int, p models.Player) error {
	if realm == "" {
		realm = "x3"
	}
	condition := "excluded.dps > leaderboard.dps"
	role := sirus.GetRoleString(p.ClassID, p.Spec)
	if role == "hps" {
		condition = "excluded.hps > leaderboard.hps"
	}
	t4 := sirus.GetT4Count(p.Itemset)

	query := fmt.Sprintf(`
		INSERT INTO leaderboard (raid_id, boss_id, class_id, spec_id, player_name, ilvl, guild_id, zodiac, category, t4, role, dps, hps, realm) 
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
		ON CONFLICT(raid_id, boss_id, class_id, spec_id, player_name) 
		DO UPDATE SET 
    		ilvl = excluded.ilvl,
			zodiac = excluded.zodiac,
			category = excluded.category,
			t4 = excluded.t4,
    		dps = excluded.dps,
    		hps = excluded.hps
		WHERE %s;
	`, condition)
	_, err := s.db.Exec(query, raid, boss, p.ClassID, p.Spec, p.Name, p.Ilvl, p.Guild.ID, p.Zodiac.ID, p.Category, t4, role, p.Dps, p.Hps, realm)
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

func (s *LeaderboardStorage) GetBossTop(realm string, raidID, bossID, guildID int, role string) ([]models.PlayerReport, error) {
	if realm == "" {
		realm = "x3"
	}
	orderBy := "dps"
	if role == "hps" {
		orderBy = "hps"
	}

	query := fmt.Sprintf(`
		SELECT l.player_name, l.class_id, l.spec_id, l.ilvl, l.zodiac, l.category, l.t4, l.dps, l.hps
		FROM leaderboard l
		JOIN guild_members gm ON l.player_name = gm.name AND COALESCE(l.realm, 'x3') = COALESCE(gm.realm, 'x3')
		WHERE l.raid_id = ? 
		  AND l.boss_id = ? 
		  AND l.role = ? 
		  AND gm.guild_id = ?
		  AND COALESCE(l.realm, 'x3') = ?
		ORDER BY l.%s DESC
		LIMIT 30`, orderBy)

	rows, err := s.db.Query(query, raidID, bossID, role, guildID, realm)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var players []models.PlayerReport
	for rows.Next() {
		var p models.PlayerReport
		err := rows.Scan(&p.Name, &p.ClassID, &p.SpecID, &p.Ilvl, &p.Zodiac, &p.Category, &p.T4, &p.Dps, &p.Hps)
		if err != nil {
			return nil, err
		}

		p.Role = sirus.GetRole(p.ClassID, p.SpecID)
		p.SpecName = sirus.GetSpecName(p.ClassID, p.SpecID)

		players = append(players, p)
	}

	return players, nil
}
