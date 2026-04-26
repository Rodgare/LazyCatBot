package storage

import (
	"database/sql"

	_ "modernc.org/sqlite"
)

type PlayerSubscribeStorage struct {
	db *sql.DB
}

func NewPlayerSubscribeStorage(db *sql.DB) *PlayerSubscribeStorage {
	return &PlayerSubscribeStorage{db: db}
}

func (s *PlayerSubscribeStorage) InitDB() error {
	query := `
    CREATE TABLE IF NOT EXISTS player_subscribe (
        id INTEGER,
		name TEXT
        channel_id TEXT,
        discord_id TEXT,
        PRIMARY KEY (id, name, channel_id, discord_id)
    );`

	_, err := s.db.Exec(query)
	return err
}

func (s *PlayerSubscribeStorage) Subscribe(id int, name string, channelID, discordID string) error {
	query := `INSERT OR REPLACE INTO character_subscribe (id, name, channel_id, discord_id) VALUES (?, ?, ?, ?)`
	_, err := s.db.Exec(query, id, name, channelID, discordID)
	return err
}

func (s *PlayerSubscribeStorage) Unsubscribe(character int, channelID, discordID string) error {
	query := `DELETE FROM character_subscribe WHERE character_id = ? AND channel_id = ? AND discord_id = ?`
	_, err := s.db.Exec(query, character, channelID, discordID)
	return err
}

func (s *PlayerSubscribeStorage) GetTrackedCharacters() (map[int][]string, error) {
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

func (s *PlayerSubscribeStorage) GetCharactersByChannel(channelID string) ([]int, error) {
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
