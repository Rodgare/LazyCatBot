package storage

import (
	"database/sql"

	_ "modernc.org/sqlite"
)

type CharacterSubscribeStorage struct {
	db *sql.DB
}

func NewCharacterSubscribeStorage(db *sql.DB) *CharacterSubscribeStorage {
	return &CharacterSubscribeStorage{db: db}
}

func (s *CharacterSubscribeStorage) InitDB() error {
	query := `
    CREATE TABLE IF NOT EXISTS character_subscribe (
        character_id INTEGER,
        channel_id TEXT,
        discord_id TEXT,
        PRIMARY KEY (character_id, channel_id, discord_id)
    );`

	_, err := s.db.Exec(query)
	return err
}

func (s *CharacterSubscribeStorage) Subscribe(character int, channelID, discordID string) error {
	query := `INSERT OR REPLACE INTO character_subscribe (character_id, channel_id, discord_id) VALUES (?, ?, ?)`
	_, err := s.db.Exec(query, character, channelID, discordID)
	return err
}

func (s *CharacterSubscribeStorage) Unsubscribe(character int, channelID, discordID string) error {
	query := `DELETE FROM character_subscribe WHERE character_id = ? AND channel_id = ? AND discord_id = ?`
	_, err := s.db.Exec(query, character, channelID, discordID)
	return err
}

func (s *CharacterSubscribeStorage) GetSubscribers(character int) ([]string, error) {
	query := `SELECT channel_id FROM character_subscribe WHERE character_id = ?`
	rows, err := s.db.Query(query, character)
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

func (s *CharacterSubscribeStorage) GetTrackedCharacters() ([]int, error) {
	query := `SELECT DISTINCT character_id FROM character_subscribe`
	rows, err := s.db.Query(query)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var characters []int
	for rows.Next() {
		var character int
		if err := rows.Scan(&character); err != nil {
			return nil, err
		}
		characters = append(characters, character)
	}

	return characters, nil
}
