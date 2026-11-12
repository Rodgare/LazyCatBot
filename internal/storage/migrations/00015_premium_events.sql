-- +goose Up
-- +goose StatementBegin
CREATE TABLE IF NOT EXISTS premium_events (
    event_key TEXT PRIMARY KEY,
    event_name TEXT NOT NULL DEFAULT '',
    created_at INTEGER NOT NULL
);
CREATE INDEX IF NOT EXISTS idx_premium_events_created ON premium_events (created_at);
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
DROP TABLE IF EXISTS premium_events;
-- +goose StatementEnd
