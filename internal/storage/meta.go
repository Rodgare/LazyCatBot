package storage

import "database/sql"

// GetAllWelcomeChannels returns the system/welcome channel of every server the
// bot has seen. These channels are used for rare bot announcements.
func (s *SubscribeStorage) GetAllWelcomeChannels() ([]string, error) {
	rows, err := s.db.Query(`SELECT DISTINCT channel_id FROM welcome_channels WHERE channel_id != ''`)
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
	return channels, rows.Err()
}

// MetaGet returns a stored key/value pair, or an empty string when missing.
func (s *SubscribeStorage) MetaGet(key string) (string, error) {
	var value string
	err := s.db.QueryRow(`SELECT value FROM bot_meta WHERE key = ?`, key).Scan(&value)
	if err == sql.ErrNoRows {
		return "", nil
	}
	return value, err
}

// MetaSet stores a key/value pair.
func (s *SubscribeStorage) MetaSet(key, value string) error {
	_, err := s.db.Exec(`INSERT OR REPLACE INTO bot_meta (key, value) VALUES (?, ?)`, key, value)
	return err
}
