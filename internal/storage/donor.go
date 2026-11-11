package storage

import (
	"database/sql"
	"strings"
	"time"

	_ "modernc.org/sqlite"
)

type DonorStorage struct {
	db *sql.DB
}

func NewDonorStorage(db *sql.DB) *DonorStorage {
	return &DonorStorage{db: db}
}

func donorKey(name string) string {
	return strings.ToLower(strings.TrimSpace(name))
}

// Upsert records a donation for a nickname. Nicknames are matched
// case-insensitively and without a realm, so a donor lights up on every realm.
func (s *DonorStorage) Upsert(name string, amount int64, currency string) error {
	key := donorKey(name)
	if key == "" {
		return nil
	}
	now := time.Now().Unix()
	_, err := s.db.Exec(`INSERT INTO donors (name_key, name, total_amount, currency, created_at, updated_at)
		VALUES (?, ?, ?, ?, ?, ?)
		ON CONFLICT(name_key) DO UPDATE SET
			name = excluded.name,
			total_amount = donors.total_amount + excluded.total_amount,
			currency = excluded.currency,
			updated_at = excluded.updated_at`,
		key, strings.TrimSpace(name), amount, strings.ToLower(strings.TrimSpace(currency)), now, now)
	return err
}

// Names returns all stored donor name keys (already lowercased).
func (s *DonorStorage) Names() ([]string, error) {
	rows, err := s.db.Query(`SELECT name_key FROM donors`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var names []string
	for rows.Next() {
		var n string
		if err := rows.Scan(&n); err != nil {
			return nil, err
		}
		names = append(names, n)
	}
	return names, rows.Err()
}
