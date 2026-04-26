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
	query := `INSERT OR REPLACE INTO player_subscribe (id, name, channel_id, discord_id) VALUES (?, ?, ?, ?)`
	_, err := s.db.Exec(query, id, name, channelID, discordID)
	return err
}

func (s *PlayerSubscribeStorage) Unsubscribe(player int, channelID, discordID string) error {
	query := `DELETE FROM player_subscribe WHERE id = ? AND channel_id = ? AND discord_id = ?`
	_, err := s.db.Exec(query, player, channelID, discordID)
	return err
}

func (s *PlayerSubscribeStorage) GetTrackedPlayers() (map[int][]string, error) {
	query := `SELECT id, channel_id FROM player_subscribe`
	rows, err := s.db.Query(query)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	res := make(map[int][]string)
	for rows.Next() {
		var player int
		var channel string
		if err := rows.Scan(&player, &channel); err != nil {
			return nil, err
		}
		res[player] = append(res[player], channel)
	}

	return res, nil
}

func (s *PlayerSubscribeStorage) GetPlayersByChannel(channelID string) ([]int, error) {
	query := `SELECT id FROM player_subscribe WHERE channel_id = ?`
	rows, err := s.db.Query(query, channelID)
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
	return players, nil
}
