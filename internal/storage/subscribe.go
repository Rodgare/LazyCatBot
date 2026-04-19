package storage

import (
	"database/sql"

	_ "modernc.org/sqlite"
)

type SubscribeStorage struct {
	db *sql.DB
}

func NewSubscribeStorage(db *sql.DB) *SubscribeStorage {
	return &SubscribeStorage{db: db}
}

func (s *SubscribeStorage) InitDB() error {
	query := `
    CREATE TABLE IF NOT EXISTS subscribe (
        guild_id INTEGER,
        channel_id TEXT,
        discord_id TEXT,
        PRIMARY KEY (guild_id, channel_id, discord_id)
    );
    CREATE INDEX IF NOT EXISTS idx_rank_lookup ON subscribe (guild_id, channel_id, discord_id);
    `

	_, err := s.db.Exec(query)
	return err
}

func (s *SubscribeStorage) Subscribe(guild int, channelID, discordID string) error {
	query := `INSERT OR REPLACE INTO subscribe (guild_id, channel_id, discord_id) VALUES (?, ?, ?)`
	_, err := s.db.Exec(query, guild, channelID, discordID)
	return err
}

func (s *SubscribeStorage) GetSubscribers(guild int) ([]string, error) {
	query := `SELECT channel_id FROM subscribe WHERE guild_id = ?`
	rows, err := s.db.Query(query, guild)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var channels []string
	for rows.Next() {
		var channelID string
		if err := rows.Scan(&channelID); err != nil {
			return nil, err
		}
		channels = append(channels, channelID)
	}

	return channels, nil
}

func (s *SubscribeStorage) GetTrackedGuilds() ([]int, error) {
	query := `SELECT DISTINCT guild_id FROM subscribe`
	rows, err := s.db.Query(query)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var guilds []int
	for rows.Next() {
		var guild int
		if err := rows.Scan(&guild); err != nil {
			return nil, err
		}
		guilds = append(guilds, guild)
	}

	return guilds, nil
}
