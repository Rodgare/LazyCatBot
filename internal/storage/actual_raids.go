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

func (s *ActualRaidsStorage) ResetActualRaids(realm string) error {
	if realm == "" {
		realm = "x3"
	}
	query := `DELETE FROM actual_raids WHERE realm = ?`
	_, err := s.db.Exec(query, realm)
	return err
}

func (s *ActualRaidsStorage) UpdateActualRaids(raidID, bossID int, realm string) error {
	if realm == "" {
		realm = "x3"
	}
	query := `INSERT OR REPLACE INTO actual_raids (raid_id, boss_id, realm) VALUES (?, ?, ?)`
	_, err := s.db.Exec(query, raidID, bossID, realm)
	return err
}

func (s *ActualRaidsStorage) GetActualRaids(realm string) ([]models.ActualRaid, error) {
	if realm == "" {
		realm = "x3"
	}
	query := `SELECT raid_id, boss_id FROM actual_raids WHERE realm = ?`
	rows, err := s.db.Query(query, realm)
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
