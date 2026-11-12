package storage

import (
	"LazyCatBot/internal/models"
	"LazyCatBot/internal/sirus"
	"database/sql"
	"fmt"
	"log/slog"
	"time"

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

func (s *LeaderboardStorage) UpdateLeaderboardStorage(realm string, raidOrder, encounter, classID, specID int, weekFrom string, players []models.LeaderboardPlayer) error {
	if realm == "" {
		realm = "x3"
	}
	if weekFrom == "" {
		weekFrom = time.Now().In(time.FixedZone("MSK", 3*3600)).Format("2006-01-02")
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

	stmt, err := tx.Prepare(`INSERT OR IGNORE INTO leaderboard (raid_id, boss_id, class_id, spec_id, player_name, ilvl, guild_id, zodiac, category, t4, set_pieces, role, dps, hps, realm) 
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`)
	if err != nil {
		return err
	}
	defer stmt.Close()

	historyStmt, err := tx.Prepare(`INSERT INTO leaderboard_history
		(realm, raid_id, boss_id, class_id, spec_id, player_name, ilvl, guild_id, zodiac, category, t4, set_pieces, role, dps, hps, snapshot_week, captured_at)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
		ON CONFLICT(realm, raid_id, boss_id, class_id, spec_id, player_name, snapshot_week)
		DO UPDATE SET
			ilvl = excluded.ilvl,
			guild_id = excluded.guild_id,
			zodiac = excluded.zodiac,
			category = excluded.category,
			t4 = excluded.t4,
			set_pieces = excluded.set_pieces,
			role = excluded.role,
			dps = MAX(leaderboard_history.dps, excluded.dps),
			hps = MAX(leaderboard_history.hps, excluded.hps),
			captured_at = excluded.captured_at`)
	if err != nil {
		return err
	}
	defer historyStmt.Close()

	role := sirus.GetRoleString(classID, specID)

	keep := make(map[string]models.LeaderboardPlayer, len(players))
	for _, p := range players {
		prev, ok := keep[p.Name]
		if !ok {
			keep[p.Name] = p
			continue
		}
		better := p.Dps > prev.Dps
		if role == "hps" {
			better = p.Hps > prev.Hps
		}
		if better {
			keep[p.Name] = p
		}
	}

	capturedAt := time.Now().Unix()
	saved := 0
	for _, p := range keep {
		t4 := sirus.GetT4CountForLeaderboard(p.Itemset)
		setPieces := sirus.GetSetPiecesForLeaderboard(p.Itemset)
		_, err = stmt.Exec(raidOrder, encounter, p.ClassID, p.SpecID, p.Name, p.Ilvl, p.GuildID, p.Zodiac.ID, p.Category, t4, setPieces, role, p.Dps, p.Hps, realm)

		if err != nil {
			return err
		}

		_, err = historyStmt.Exec(realm, raidOrder, encounter, p.ClassID, p.SpecID, p.Name, p.Ilvl, p.GuildID,
			p.Zodiac.ID, p.Category, t4, setPieces, role, p.Dps, p.Hps, weekFrom, capturedAt)
		if err != nil {
			return err
		}
		saved++
	}

	err = tx.Commit()
	if err == nil {
		s.logger.Info("[DB] Successfully saved",
			"realm", realm, "players_count", saved, "parses", len(players), "duplicates", len(players)-saved,
			"raid_id", raidOrder, "boss_id", encounter, "class_id", classID, "spec_id", specID, "week_from", weekFrom)
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
	setPieces := sirus.GetSetPieces(p.Itemset)

	query := fmt.Sprintf(`
		INSERT INTO leaderboard (raid_id, boss_id, class_id, spec_id, player_name, ilvl, guild_id, zodiac, category, t4, set_pieces, role, dps, hps, realm) 
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
		ON CONFLICT(raid_id, boss_id, class_id, spec_id, player_name, realm) 
		DO UPDATE SET 
    		ilvl = excluded.ilvl,
			zodiac = excluded.zodiac,
			category = excluded.category,
			t4 = excluded.t4,
			set_pieces = excluded.set_pieces,
    		dps = excluded.dps,
    		hps = excluded.hps
		WHERE %s;
	`, condition)
	_, err := s.db.Exec(query, raid, boss, p.ClassID, p.Spec, p.Name, p.Ilvl, p.Guild.ID, p.Zodiac.ID, p.Category, t4, setPieces, role, p.Dps, p.Hps, realm)
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

func (s *LeaderboardStorage) IsLeaderboardSyncDone(realm, syncDate string, raid, boss, class, spec int) (bool, error) {
	var done bool
	err := s.db.QueryRow("SELECT EXISTS(SELECT 1 FROM leaderboard_sync_state WHERE realm=? AND sync_date=? AND raid_id=? AND boss_id=? AND class_id=? AND spec_id=?)",
		realm, syncDate, raid, boss, class, spec).Scan(&done)
	return done, err
}

func (s *LeaderboardStorage) MarkLeaderboardSyncDone(realm, syncDate string, raid, boss, class, spec int) error {
	_, err := s.db.Exec("INSERT OR IGNORE INTO leaderboard_sync_state (realm, sync_date, raid_id, boss_id, class_id, spec_id) VALUES (?, ?, ?, ?, ?, ?)",
		realm, syncDate, raid, boss, class, spec)
	return err
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
		SELECT l.player_name, l.class_id, l.spec_id,
		       COALESCE(l.ilvl, 0), COALESCE(l.zodiac, 0), COALESCE(l.category, 0), COALESCE(l.t4, 0),
		       COALESCE(l.set_pieces, ''), COALESCE(l.dps, 0), COALESCE(l.hps, 0),
		       COALESCE(cc.black_diamonds, 0), COALESCE(cc.title, '')
		FROM leaderboard l
		JOIN guild_members gm ON l.player_name = gm.name AND COALESCE(l.realm, 'x3') = COALESCE(gm.realm, 'x3')
		LEFT JOIN character_cache cc ON cc.realm = COALESCE(l.realm, 'x3') AND cc.name = l.player_name
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
		err := rows.Scan(&p.Name, &p.ClassID, &p.SpecID, &p.Ilvl, &p.Zodiac, &p.Category, &p.T4, &p.SetPieces, &p.Dps, &p.Hps, &p.BlackDiamonds, &p.Title)
		if err != nil {
			return nil, err
		}

		p.Role = sirus.GetRole(p.ClassID, p.SpecID)
		p.SpecName = sirus.GetSpecName(p.ClassID, p.SpecID)

		players = append(players, p)
	}

	return players, nil
}

// LeaderboardRange selects the time window for a guild ranking.
type LeaderboardRange string

const (
	Range2W  LeaderboardRange = "2w"
	Range4W  LeaderboardRange = "4w"
	RangeAll LeaderboardRange = "all"
)

func (r LeaderboardRange) Valid() bool {
	switch r {
	case Range2W, Range4W, RangeAll:
		return true
	}
	return false
}

func (r LeaderboardRange) Title() string {
	switch r {
	case Range4W:
		return "4 КД"
	case RangeAll:
		return "всё время"
	default:
		return "2 КД"
	}
}

// GetBossTopRange returns the guild ranking for a time window. 2 KD is served
// from the current snapshot (historical behaviour). 4 KD and all-time are
// aggregated from leaderboard_history, which accumulates one best-per-week row
// per player as the nightly sync runs, so no extra Sirus API calls are needed.
func (s *LeaderboardStorage) GetBossTopRange(realm string, raidID, bossID, guildID int, role string, rng LeaderboardRange) ([]models.PlayerReport, error) {
	if rng == "" || rng == Range2W {
		return s.GetBossTop(realm, raidID, bossID, guildID, role)
	}
	if realm == "" {
		realm = "x3"
	}

	orderBy := "dps"
	if role == "hps" {
		orderBy = "hps"
	}

	where := fmt.Sprintf(`h.realm = ? AND h.raid_id = ? AND h.boss_id = ? AND h.role = ? AND h.%s > 0 AND gm.guild_id = ?`, orderBy)
	args := []any{realm, raidID, bossID, role, guildID}
	if rng == Range4W {
		where += " AND h.captured_at >= ?"
		args = append(args, time.Now().AddDate(0, 0, -28).Unix())
	}

	query := fmt.Sprintf(`
		SELECT x.player_name, x.class_id, x.spec_id,
		       COALESCE(x.ilvl, 0), COALESCE(x.zodiac, 0), COALESCE(x.category, 0), COALESCE(x.t4, 0),
		       COALESCE(x.set_pieces, ''), COALESCE(x.dps, 0), COALESCE(x.hps, 0),
		       COALESCE(cc.black_diamonds, 0), COALESCE(cc.title, '')
		FROM (
			SELECT h.*, ROW_NUMBER() OVER (PARTITION BY h.player_name ORDER BY h.%s DESC) AS rn
			FROM leaderboard_history h
			JOIN guild_members gm ON gm.name = h.player_name AND COALESCE(gm.realm, 'x3') = COALESCE(h.realm, 'x3')
			WHERE %s
		) x
		LEFT JOIN character_cache cc ON cc.realm = COALESCE(x.realm, 'x3') AND cc.name = x.player_name
		WHERE x.rn = 1
		ORDER BY x.%s DESC
		LIMIT 30`, orderBy, where, orderBy)

	rows, err := s.db.Query(query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var players []models.PlayerReport
	for rows.Next() {
		var p models.PlayerReport
		if err := rows.Scan(&p.Name, &p.ClassID, &p.SpecID, &p.Ilvl, &p.Zodiac, &p.Category, &p.T4, &p.SetPieces, &p.Dps, &p.Hps, &p.BlackDiamonds, &p.Title); err != nil {
			return nil, err
		}
		p.Role = sirus.GetRole(p.ClassID, p.SpecID)
		p.SpecName = sirus.GetSpecName(p.ClassID, p.SpecID)
		players = append(players, p)
	}
	return players, rows.Err()
}
