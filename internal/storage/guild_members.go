package storage

import (
	"database/sql"

	_ "modernc.org/sqlite"
)

type GuildMembersStorage struct {
	db *sql.DB
}

func NewGuildMembersStorage(db *sql.DB) *GuildMembersStorage {
	return &GuildMembersStorage{db: db}
}

func (s *GuildMembersStorage) InitDB() error {
	query := `
	CREATE TABLE IF NOT EXISTS guild_members (
	id INTEGER,
	guild_id INTEGER,
	PRIMARY KEY (id, guild_id)
	);
	CREATE INDEX IF NOT EXISTS idx_guild ON guild_members (guild_id);
	`

	_, err := s.db.Exec(query)
	return err
}
