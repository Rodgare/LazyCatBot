-- +goose Up
-- +goose StatementBegin
CREATE TABLE IF NOT EXISTS character_cache (
    realm TEXT NOT NULL,
    name TEXT NOT NULL,
    title TEXT DEFAULT '',
    mythic_rating REAL DEFAULT 0,
    black_diamonds INTEGER DEFAULT 0,
    updated_at INTEGER DEFAULT 0,
    PRIMARY KEY (realm, name)
);
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
DROP TABLE IF EXISTS character_cache;
-- +goose StatementEnd