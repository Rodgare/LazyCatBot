package storage

import (
	"LazyCatBot/internal/models"
	"database/sql"
)

type ActualRaidsStorage struct {
	db *sql.DB
}

func NewActualRaidsStorage(db *sql.DB) *ActualRaidsStorage {
	return &ActualRaidsStorage{db: db}
}

func (s *ActualRaidsStorage) ResetActualRaids(serverID int) error {
	query := `
	DELETE FROM actual_raids
	WHERE server_id = ?
	`

	_, err := s.db.Exec(query, serverID)
	return err
}

func (s *ActualRaidsStorage) UpdateActualRaids(raidID, bossID, serverID int) error {
	query := `
	INSERT OR REPLACE INTO actual_raids (raid_id, boss_id, server_id)
	VALUES (?, ?, ?)
	`

	_, err := s.db.Exec(query, raidID, bossID, serverID)
	return err
}

func (s *ActualRaidsStorage) GetActualRaids(serverID int) ([]models.ActualRaid, error) {
	query := `
	SELECT raid_id, boss_id
	FROM actual_raids
	WHERE server_id = ?
	`

	rows, err := s.db.Query(query, serverID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var raids []models.ActualRaid
	for rows.Next() {
		var raid models.ActualRaid
		err := rows.Scan(&raid.RaidID, &raid.BossID)
		if err != nil {
			return nil, err
		}
		raids = append(raids, raid)
	}

	return raids, nil
}
