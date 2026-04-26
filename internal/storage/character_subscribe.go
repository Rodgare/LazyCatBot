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

func (s *CharacterSubscribeStorage) GetTrackedCharacters() (map[int][]string, error) {
	query := `SELECT character_id, channel_id FROM character_subscribe`
	rows, err := s.db.Query(query)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	res := make(map[int][]string)
	for rows.Next() {
		var character int
		var channel string
		if err := rows.Scan(&character, &channel); err != nil {
			return nil, err
		}
		res[character] = append(res[character], channel)
	}

	return res, nil
}

func (s *CharacterSubscribeStorage) GetCharactersByChannel(channelID string) ([]int, error) {
	query := `SELECT character_id FROM character_subscribe WHERE channel_id = ?`
	rows, err := s.db.Query(query, channelID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var characters []int
	for rows.Next() {
		var id int
		if err := rows.Scan(&id); err != nil {
			return nil, err
		}
		characters = append(characters, id)
	}
	return characters, nil
}
