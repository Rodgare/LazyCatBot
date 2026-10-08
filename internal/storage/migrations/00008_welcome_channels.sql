-- +goose Up
-- +goose StatementBegin
CREATE TABLE IF NOT EXISTS welcome_channels (
    discord_id TEXT PRIMARY KEY,
    channel_id TEXT NOT NULL
);
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
DROP TABLE IF EXISTS welcome_channels;
-- +goose StatementEnd