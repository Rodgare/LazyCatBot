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

type TrackedPlayerKey struct {
	PlayerID int
	Realm    string
}

func (s *PlayerSubscribeStorage) Subscribe(id int, name string, channelID, discordID, realm string) error {
	if realm == "" {
		realm = "x3"
	}
	query := `INSERT OR REPLACE INTO player_subscribe (id, name, channel_id, discord_id, realm) VALUES (?, ?, ?, ?, ?)`
	_, err := s.db.Exec(query, id, name, channelID, discordID, realm)
	return err
}

func (s *PlayerSubscribeStorage) Unsubscribe(player int, channelID, discordID string) error {
	query := `DELETE FROM player_subscribe WHERE id = ? AND channel_id = ? AND discord_id = ?`
	_, err := s.db.Exec(query, player, channelID, discordID)
	return err
}

func (s *PlayerSubscribeStorage) GetTrackedPlayers() (map[TrackedPlayerKey][]string, error) {
	query := `SELECT id, channel_id, COALESCE(realm, 'x3') FROM player_subscribe`
	rows, err := s.db.Query(query)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	res := make(map[TrackedPlayerKey][]string)
	for rows.Next() {
		var player int
		var channel string
		var realm string
		if err := rows.Scan(&player, &channel, &realm); err != nil {
			return nil, err
		}
		key := TrackedPlayerKey{PlayerID: player, Realm: realm}
		res[key] = append(res[key], channel)
	}

	return res, nil
}

func (s *PlayerSubscribeStorage) GetPlayersByChannel(channelID string) (map[int]string, error) {
	query := `SELECT id, name FROM player_subscribe WHERE channel_id = ?`
	rows, err := s.db.Query(query, channelID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	players := make(map[int]string)

	for rows.Next() {
		var id int
		var name string
		if err := rows.Scan(&id, &name); err != nil {
			return nil, err
		}
		players[id] = name
	}
	return players, nil
}
