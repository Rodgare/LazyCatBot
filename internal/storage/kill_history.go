package storage

import (
	"database/sql"
	"fmt"
	"log/slog"
	"strings"
	"time"

	"LazyCatBot/internal/models"
	"LazyCatBot/internal/sirus"

	_ "modernc.org/sqlite"
)

// KillHistoryStorage persists enriched kill reports so attendance, weekly
// digests and player profiles can be built without new Sirus API calls.
type KillHistoryStorage struct {
	db     *sql.DB
	logger *slog.Logger
}

func NewKillHistoryStorage(db *sql.DB, logger *slog.Logger) *KillHistoryStorage {
	return &KillHistoryStorage{db: db, logger: logger}
}

// RecordKill stores one kill and its roster. It is idempotent: a repeated kill
// replaces the roster instead of duplicating rows.
func (s *KillHistoryStorage) RecordKill(report models.BossKillReport) error {
	if report.GuildID == 0 || report.KillID == 0 {
		return nil
	}
	realm := report.Realm
	if realm == "" {
		realm = "x3"
	}
	now := time.Now().Unix()
	killsAt := parseKillTime(report.KilledAt)

	tx, err := s.db.Begin()
	if err != nil {
		return err
	}
	defer tx.Rollback()

	_, err = tx.Exec(`INSERT INTO kill_history
		(realm, kill_id, guild_id, raid_id, boss_id, kills_at, duration, attempts, total_dps, total_hps, created_at)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
		ON CONFLICT(realm, kill_id, guild_id) DO UPDATE SET
			raid_id = excluded.raid_id,
			boss_id = excluded.boss_id,
			kills_at = excluded.kills_at,
			duration = excluded.duration,
			attempts = excluded.attempts,
			total_dps = excluded.total_dps,
			total_hps = excluded.total_hps`,
		realm, report.KillID, report.GuildID, report.RaidOrder, report.BossID, killsAt,
		report.Duration, report.Attempts, report.TotalDps, report.TotalHps, now)
	if err != nil {
		return err
	}

	if _, err := tx.Exec(`DELETE FROM kill_players WHERE realm = ? AND kill_id = ? AND guild_id = ?`,
		realm, report.KillID, report.GuildID); err != nil {
		return err
	}

	stmt, err := tx.Prepare(`INSERT INTO kill_players
		(realm, kill_id, guild_id, player_name, guid, class_id, spec_id, role, dps, hps, ilvl, t4, set_pieces)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`)
	if err != nil {
		return err
	}
	defer stmt.Close()

	for _, p := range report.Players {
		role := "dps"
		if p.Role == 1 {
			role = "hps"
		}
		if _, err := stmt.Exec(realm, report.KillID, report.GuildID, p.Name, p.GUID,
			p.ClassID, p.SpecID, role, p.Dps, p.Hps, p.Ilvl, p.T4, p.SetPieces); err != nil {
			return err
		}
	}

	if err := tx.Commit(); err != nil {
		return err
	}
	if s.logger != nil {
		s.logger.Info("[DB] Kill history saved",
			"realm", realm, "kill_id", report.KillID, "guild_id", report.GuildID,
			"players", len(report.Players))
	}
	return nil
}

// GetAttendance returns the guild's total kills and per-player attendance in a
// window starting at since (unix seconds). Only actual raid encounters are
// counted so old/irrelevant raids do not skew attendance percentages.
func (s *KillHistoryStorage) GetAttendance(realm string, guildID int, since int64) (int, []models.AttendanceEntry, error) {
	if realm == "" {
		realm = "x3"
	}

	var total int
	if err := s.db.QueryRow(`SELECT COUNT(*)
		FROM kill_history h
		JOIN actual_raids ar ON ar.realm = h.realm AND ar.raid_id = h.raid_id AND ar.boss_id = h.boss_id
		WHERE h.realm = ? AND h.guild_id = ? AND h.kills_at >= ?`,
		realm, guildID, since).Scan(&total); err != nil {
		return 0, nil, err
	}
	if total == 0 {
		return 0, nil, nil
	}

	rows, err := s.db.Query(`
		WITH actual AS (
			SELECT h.realm, h.kill_id, h.guild_id, h.kills_at
			FROM kill_history h
			JOIN actual_raids ar ON ar.realm = h.realm AND ar.raid_id = h.raid_id AND ar.boss_id = h.boss_id
			WHERE h.realm = ? AND h.guild_id = ? AND h.kills_at >= ?
		),
		agg AS (
			SELECT p.player_name, p.class_id, p.spec_id, p.role,
			       COUNT(DISTINCT p.kill_id) AS attended,
			       COALESCE(MAX(a.kills_at), 0) AS last_seen,
			       COALESCE(MAX(p.ilvl), 0) AS ilvl,
			       COALESCE(MAX(cc.black_diamonds), 0) AS black_diamonds,
			       COALESCE(MAX(cc.title), '') AS title
			FROM kill_players p
			JOIN actual a ON a.realm = p.realm AND a.kill_id = p.kill_id AND a.guild_id = p.guild_id
			LEFT JOIN character_cache cc ON cc.realm = p.realm AND cc.name = p.player_name
			GROUP BY p.player_name
		),
		latest AS (
			SELECT p.player_name, p.set_pieces,
			       ROW_NUMBER() OVER (PARTITION BY p.player_name ORDER BY a.kills_at DESC) AS rn
			FROM kill_players p
			JOIN actual a ON a.realm = p.realm AND a.kill_id = p.kill_id AND a.guild_id = p.guild_id
		)
		SELECT agg.player_name, agg.class_id, agg.spec_id, agg.role, agg.attended,
		       agg.last_seen, agg.ilvl, agg.black_diamonds, agg.title,
		       COALESCE(latest.set_pieces, '')
		FROM agg
		LEFT JOIN latest ON latest.player_name = agg.player_name AND latest.rn = 1
		ORDER BY agg.attended DESC, agg.player_name ASC`, realm, guildID, since)
	if err != nil {
		return 0, nil, err
	}
	defer rows.Close()

	var entries []models.AttendanceEntry
	for rows.Next() {
		var e models.AttendanceEntry
		if err := rows.Scan(&e.Name, &e.ClassID, &e.SpecID, &e.Role, &e.Kills, &e.LastSeen, &e.Ilvl,
			&e.BlackDiamonds, &e.Title, &e.SetPieces); err != nil {
			return 0, nil, err
		}
		entries = append(entries, e)
	}
	return total, entries, rows.Err()
}

// GuildPlayerNames returns every player with a kill in an actual raid inside
// the window. Used by /profile autocomplete and case-insensitive lookup.
func (s *KillHistoryStorage) GuildPlayerNames(realm string, guildID int, since int64) ([]string, error) {
	if realm == "" {
		realm = "x3"
	}
	rows, err := s.db.Query(`
		SELECT DISTINCT p.player_name
		FROM kill_players p
		JOIN kill_history h ON h.realm = p.realm AND h.kill_id = p.kill_id AND h.guild_id = p.guild_id
		JOIN actual_raids ar ON ar.realm = h.realm AND ar.raid_id = h.raid_id AND ar.boss_id = h.boss_id
		WHERE p.realm = ? AND p.guild_id = ? AND h.kills_at >= ?
		ORDER BY p.player_name`, realm, guildID, since)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var names []string
	for rows.Next() {
		var name string
		if err := rows.Scan(&name); err != nil {
			return nil, err
		}
		names = append(names, name)
	}
	return names, rows.Err()
}

// SearchGuildPlayerNames filters guild players by a case-insensitive prefix.
func (s *KillHistoryStorage) SearchGuildPlayerNames(realm string, guildID int, prefix string, since int64, limit int) ([]string, error) {
	if limit <= 0 || limit > 25 {
		limit = 25
	}
	names, err := s.GuildPlayerNames(realm, guildID, since)
	if err != nil {
		return nil, err
	}

	prefix = strings.ToLower(strings.TrimSpace(prefix))
	out := make([]string, 0, limit)
	for _, name := range names {
		if prefix != "" && !strings.HasPrefix(strings.ToLower(name), prefix) {
			continue
		}
		out = append(out, name)
		if len(out) == limit {
			break
		}
	}
	return out, nil
}

// resolveGuildPlayerName maps a user-entered name to the exact stored spelling,
// ignoring case (SQLite NOCASE does not cover Cyrillic).
func (s *KillHistoryStorage) resolveGuildPlayerName(realm string, guildID int, name string, since int64) (string, error) {
	names, err := s.GuildPlayerNames(realm, guildID, since)
	if err != nil {
		return name, err
	}
	for _, candidate := range names {
		if strings.EqualFold(candidate, name) {
			return candidate, nil
		}
	}
	return name, nil
}

// GetWeeklyDigest assembles the data for the weekly image. Both bounds are unix
// seconds; since is inclusive and until exclusive.
func (s *KillHistoryStorage) GetWeeklyDigest(realm string, guildID int, since, until int64) (*models.WeeklyDigest, error) {
	if realm == "" {
		realm = "x3"
	}

	digest := &models.WeeklyDigest{Realm: realm, Since: since, Until: until}

	if err := s.db.QueryRow(`SELECT COUNT(*)
		FROM kill_history h
		JOIN actual_raids ar ON ar.realm = h.realm AND ar.raid_id = h.raid_id AND ar.boss_id = h.boss_id
		WHERE h.realm = ? AND h.guild_id = ? AND h.kills_at >= ? AND h.kills_at < ?`,
		realm, guildID, since, until).Scan(&digest.TotalKills); err != nil {
		return nil, err
	}

	firstRows, err := s.db.Query(`
		SELECT h.boss_id, h.raid_id, MIN(h.kills_at) AS first_kill
		FROM kill_history h
		JOIN actual_raids ar ON ar.realm = h.realm AND ar.raid_id = h.raid_id AND ar.boss_id = h.boss_id
		WHERE h.realm = ? AND h.guild_id = ?
		GROUP BY h.boss_id, h.raid_id
		HAVING MIN(h.kills_at) >= ? AND MIN(h.kills_at) < ?
		ORDER BY first_kill ASC`, realm, guildID, since, until)
	if err != nil {
		return nil, err
	}
	for firstRows.Next() {
		var fk models.FirstKill
		var raidID int
		if err := firstRows.Scan(&fk.BossID, &raidID, &fk.KilledAt); err != nil {
			firstRows.Close()
			return nil, err
		}
		fk.BossName = sirus.GetBossName(raidID, fk.BossID)
		digest.FirstKills = append(digest.FirstKills, fk)
	}
	firstRows.Close()

	if digest.TopDps, err = s.topPlayers(realm, guildID, since, until, "dps"); err != nil {
		return nil, err
	}
	if digest.TopHps, err = s.topPlayers(realm, guildID, since, until, "hps"); err != nil {
		return nil, err
	}
	if _, digest.Attendance, err = s.GetAttendance(realm, guildID, since); err != nil {
		return nil, err
	}

	return digest, nil
}

func (s *KillHistoryStorage) topPlayers(realm string, guildID int, since, until int64, role string) ([]models.DigestPlayer, error) {
	valueCol := "p.dps"
	if role == "hps" {
		valueCol = "p.hps"
	}
	query := fmt.Sprintf(`
		SELECT p.player_name, p.class_id, p.spec_id, MAX(%s) AS best,
		       COALESCE(MAX(cc.black_diamonds), 0) AS black_diamonds,
		       COALESCE(MAX(cc.title), '') AS title
		FROM kill_players p
		JOIN kill_history h ON h.realm = p.realm AND h.kill_id = p.kill_id AND h.guild_id = p.guild_id
		JOIN actual_raids ar ON ar.realm = h.realm AND ar.raid_id = h.raid_id AND ar.boss_id = h.boss_id
		LEFT JOIN character_cache cc ON cc.realm = p.realm AND cc.name = p.player_name
		WHERE p.realm = ? AND p.guild_id = ? AND h.kills_at >= ? AND h.kills_at < ?
		  AND p.role = ? AND %s > 0
		GROUP BY p.player_name
		ORDER BY best DESC
		LIMIT 5`, valueCol, valueCol)

	rows, err := s.db.Query(query, realm, guildID, since, until, role)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var players []models.DigestPlayer
	for rows.Next() {
		var p models.DigestPlayer
		if err := rows.Scan(&p.Name, &p.ClassID, &p.SpecID, &p.Value, &p.BlackDiamonds, &p.Title); err != nil {
			return nil, err
		}
		players = append(players, p)
	}
	return players, rows.Err()
}

// GetPlayerProfile returns kill-history stats for one player inside a window.
func (s *KillHistoryStorage) GetPlayerProfile(realm string, guildID int, name string, since int64) (*models.PlayerKillProfile, error) {
	if realm == "" {
		realm = "x3"
	}

	name, err := s.resolveGuildPlayerName(realm, guildID, name, since)
	if err != nil {
		return nil, err
	}

	profile := &models.PlayerKillProfile{Name: name}
	err = s.db.QueryRow(`
		SELECT COUNT(DISTINCT p.kill_id), COALESCE(MAX(p.dps), 0), COALESCE(MAX(p.hps), 0),
		       COALESCE(MIN(h.kills_at), 0), COALESCE(MAX(h.kills_at), 0), COALESCE(MAX(p.ilvl), 0)
		FROM kill_players p
		JOIN kill_history h ON h.realm = p.realm AND h.kill_id = p.kill_id AND h.guild_id = p.guild_id
		JOIN actual_raids ar ON ar.realm = h.realm AND ar.raid_id = h.raid_id AND ar.boss_id = h.boss_id
		WHERE p.realm = ? AND p.guild_id = ? AND p.player_name = ? AND h.kills_at >= ?`,
		realm, guildID, name, since).
		Scan(&profile.TotalKills, &profile.BestDps, &profile.BestHps,
			&profile.FirstSeen, &profile.LastSeen, &profile.Ilvl)
	if err != nil {
		return nil, err
	}
	if profile.TotalKills == 0 {
		return profile, nil
	}

	err = s.db.QueryRow(`
		SELECT p.guid, p.class_id, p.spec_id, p.role, p.t4, p.set_pieces FROM kill_players p
		JOIN kill_history h ON h.realm = p.realm AND h.kill_id = p.kill_id AND h.guild_id = p.guild_id
		JOIN actual_raids ar ON ar.realm = h.realm AND ar.raid_id = h.raid_id AND ar.boss_id = h.boss_id
		WHERE p.realm = ? AND p.guild_id = ? AND p.player_name = ? AND h.kills_at >= ?
		ORDER BY h.kills_at DESC LIMIT 1`,
		realm, guildID, name, since).
		Scan(&profile.GUID, &profile.ClassID, &profile.SpecID, &profile.Role, &profile.T4, &profile.SetPieces)
	if err != nil && err != sql.ErrNoRows {
		return nil, err
	}

	var guildKills int
	if err := s.db.QueryRow(`SELECT COUNT(*)
		FROM kill_history h
		JOIN actual_raids ar ON ar.realm = h.realm AND ar.raid_id = h.raid_id AND ar.boss_id = h.boss_id
		WHERE h.realm = ? AND h.guild_id = ? AND h.kills_at >= ?`,
		realm, guildID, since).Scan(&guildKills); err != nil {
		return nil, err
	}
	if guildKills > 0 {
		profile.AttendedPct = profile.TotalKills * 100 / guildKills
		if profile.AttendedPct > 100 {
			profile.AttendedPct = 100
		}
	}
	return profile, nil
}

func parseKillTime(s string) int64 {
	loc := time.FixedZone("MSK", 3*3600)
	if t, err := time.ParseInLocation("2006-01-02 15:04:05", s, loc); err == nil {
		return t.Unix()
	}
	return time.Now().Unix()
}
