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
