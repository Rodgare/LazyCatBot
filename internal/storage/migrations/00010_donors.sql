-- +goose Up
-- +goose StatementBegin
CREATE TABLE IF NOT EXISTS donors (
    name_key TEXT PRIMARY KEY,
    name TEXT NOT NULL,
    total_amount INTEGER DEFAULT 0,
    currency TEXT DEFAULT '',
    created_at INTEGER DEFAULT 0,
    updated_at INTEGER DEFAULT 0
);
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
DROP TABLE IF EXISTS donors;
-- +goose StatementEnd
