package storage

import (
	"LazyCatBot/internal/models"
	"database/sql"
	"time"

	_ "modernc.org/sqlite"
)

type CharacterCacheStorage struct {
	db *sql.DB
}

func NewCharacterCacheStorage(db *sql.DB) *CharacterCacheStorage {
	return &CharacterCacheStorage{db: db}
}

func (s *CharacterCacheStorage) Upsert(realm, name string, data models.CharacterData) error {
	if realm == "" {
		realm = "x3"
	}
	_, err := s.db.Exec(`INSERT INTO character_cache (realm, name, title, mythic_rating, black_diamonds, updated_at)
		VALUES (?, ?, ?, ?, ?, ?)
		ON CONFLICT(realm, name) DO UPDATE SET
			title = excluded.title,
			mythic_rating = excluded.mythic_rating,
			black_diamonds = excluded.black_diamonds,
			updated_at = excluded.updated_at`,
		realm, name, data.Title(), data.Challenge.CurrentScore, data.CountBlackDiamonds(), time.Now().Unix())
	return err
}

func (s *CharacterCacheStorage) Get(realm, name string) (models.CharacterCache, error) {
	var c models.CharacterCache
	err := s.db.QueryRow(`SELECT realm, name, COALESCE(title, ''), mythic_rating, black_diamonds, updated_at
		FROM character_cache WHERE realm = ? AND name = ?`, realm, name).
		Scan(&c.Realm, &c.Name, &c.Title, &c.MythicRating, &c.BlackDiamonds, &c.UpdatedAt)
	return c, err
}

func (s *CharacterCacheStorage) StaleOrMissing(realm, name string, ttlSeconds int64) (bool, error) {
	var updated int64
	err := s.db.QueryRow(`SELECT updated_at FROM character_cache WHERE realm = ? AND name = ?`, realm, name).Scan(&updated)
	if err == sql.ErrNoRows {
		return true, nil
	}
	if err != nil {
		return false, err
	}
	return time.Now().Unix()-updated > ttlSeconds, nil
}