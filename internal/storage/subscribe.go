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

func (s *SubscribeStorage) Subscribe(guild int, channelID, discordID string) error {
	query := `INSERT OR REPLACE INTO subscribe (guild_id, channel_id, discord_id) VALUES (?, ?, ?)`
	_, err := s.db.Exec(query, guild, channelID, discordID)
	return err
}

func (s *SubscribeStorage) ToggleReports(guild int, channelID string) (bool, error) {
	var current int
	err := s.db.QueryRow("SELECT is_send FROM subscribe WHERE guild_id = ? AND channel_id = ?", guild, channelID).Scan(&current)
	if err != nil {
		return false, err
	}
	newVal := 0
	if current == 0 {
		newVal = 1
	}
	_, err = s.db.Exec("UPDATE subscribe SET is_send = ? WHERE guild_id = ? AND channel_id = ?", newVal, guild, channelID)
	return newVal == 1, err
}

func (s *SubscribeStorage) IsReportsEnabled(guild int, channelID string) bool {
	var enabled int
	err := s.db.QueryRow("SELECT is_send FROM subscribe WHERE guild_id = ? AND channel_id = ?", guild, channelID).Scan(&enabled)
	if err != nil {
		return true
	}
	return enabled == 1
}

func (s *SubscribeStorage) Unsubscribe(guild int, channelID, discordID string) error {
	query := `DELETE FROM subscribe WHERE guild_id = ? AND channel_id = ? AND discord_id = ?`
	_, err := s.db.Exec(query, guild, channelID, discordID)
	return err
}

func (s *SubscribeStorage) GetChannels(guild int) ([]string, error) {
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

func (s *SubscribeStorage) GetGuildsByChannel(channelID string) ([]int, error) {
	query := `SELECT guild_id FROM subscribe WHERE channel_id = ?`
	rows, err := s.db.Query(query, channelID)
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

func (s *SubscribeStorage) GetTrackedGuilds() (map[int][]string, error) {
	query := `SELECT guild_id, channel_id FROM subscribe`
	rows, err := s.db.Query(query)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	res := make(map[int][]string)
	for rows.Next() {
		var guild int
		var channel string
		if err := rows.Scan(&guild, &channel); err != nil {
			return nil, err
		}
		res[guild] = append(res[guild], channel)
	}

	return res, nil
}

func (s *SubscribeStorage) IsDiscordGuildSubscribed(discordID string) bool {
	var exists bool
	query := `SELECT EXISTS(SELECT 1 FROM subscribe WHERE discord_id = ?)`
	s.db.QueryRow(query, discordID).Scan(&exists)
	return exists
}

func (s *SubscribeStorage) IsKillProcessed(id int, ch string) bool {
	var exists bool
	s.db.QueryRow("SELECT EXISTS(SELECT 1 FROM processed_kills WHERE kill_id=? AND channel_id=?)", id, ch).
		Scan(&exists)
	return exists
}

func (s *SubscribeStorage) MarkKillProcessed(id int, ch string) error {
	_, err := s.db.Exec("INSERT OR IGNORE INTO processed_kills (kill_id, channel_id) VALUES (?, ?)", id, ch)
	return err
}
