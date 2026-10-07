package storage

import (
	"LazyCatBot/internal/models"
	"database/sql"

	_ "modernc.org/sqlite"
)

type GuildMembersStorage struct {
	db *sql.DB
}

func NewGuildMembersStorage(db *sql.DB) *GuildMembersStorage {
	return &GuildMembersStorage{db: db}
}

func (s *GuildMembersStorage) UpdateGuildMembers(realm string, guildID int, members []models.GuildMembers) error {
	if realm == "" {
		realm = "x3"
	}
	tx, err := s.db.Begin()
	if err != nil {
		return err
	}
	defer tx.Rollback()

	_, err = tx.Exec("DELETE FROM guild_members WHERE guild_id = ? AND realm = ?", guildID, realm)
	if err != nil {
		return err
	}

	stmt, err := tx.Prepare(`INSERT INTO guild_members (id, name, ilvl, guild_id, realm) VALUES (?, ?, ?, ?, ?)`)
	if err != nil {
		return err
	}
	defer stmt.Close()

	for _, m := range members {
		_, err = stmt.Exec(m.GUID, m.Name, m.Ilvl, guildID, realm)
		if err != nil {
			return err
		}
	}

	return tx.Commit()
}

func (s *GuildMembersStorage) GetPlayersByGuildID(realm string, guildID int) ([]int, error) {
	if realm == "" {
		realm = "x3"
	}

	query := `SELECT id FROM guild_members WHERE guild_id = ? AND realm = ?`
	rows, err := s.db.Query(query, guildID, realm)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var players []int
	for rows.Next() {
		var id int
		if err := rows.Scan(&id); err != nil {
			return nil, err
		}
		players = append(players, id)
	}
	return players, rows.Err()
}

type TrackedMemberName struct {
	Name  string
	Realm string
}

func (s *GuildMembersStorage) GetAllMembers() ([]TrackedMemberName, error) {
	query := `SELECT DISTINCT name, COALESCE(realm, 'x3') FROM guild_members`
	rows, err := s.db.Query(query)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var members []TrackedMemberName
	for rows.Next() {
		var m TrackedMemberName
		if err := rows.Scan(&m.Name, &m.Realm); err != nil {
			return nil, err
		}
		members = append(members, m)
	}
	return members, rows.Err()
}
