-- +goose Up
-- +goose StatementBegin
CREATE TABLE IF NOT EXISTS kill_history (
    realm TEXT NOT NULL,
    kill_id INTEGER NOT NULL,
    guild_id INTEGER NOT NULL,
    raid_id INTEGER NOT NULL,
    boss_id INTEGER NOT NULL,
    kills_at INTEGER NOT NULL,
    duration TEXT NOT NULL DEFAULT '',
    attempts INTEGER NOT NULL DEFAULT 0,
    total_dps INTEGER NOT NULL DEFAULT 0,
    total_hps INTEGER NOT NULL DEFAULT 0,
    created_at INTEGER NOT NULL,
    PRIMARY KEY (realm, kill_id, guild_id)
);
CREATE INDEX IF NOT EXISTS idx_kill_history_guild ON kill_history (realm, guild_id, kills_at);

CREATE TABLE IF NOT EXISTS kill_players (
    realm TEXT NOT NULL,
    kill_id INTEGER NOT NULL,
    guild_id INTEGER NOT NULL,
    player_name TEXT NOT NULL,
    guid INTEGER NOT NULL DEFAULT 0,
    class_id INTEGER NOT NULL DEFAULT 0,
    spec_id INTEGER NOT NULL DEFAULT 0,
    role TEXT NOT NULL DEFAULT '',
    dps INTEGER NOT NULL DEFAULT 0,
    hps INTEGER NOT NULL DEFAULT 0,
    ilvl INTEGER NOT NULL DEFAULT 0,
    PRIMARY KEY (realm, kill_id, guild_id, player_name)
);
CREATE INDEX IF NOT EXISTS idx_kill_players_guild ON kill_players (realm, guild_id, player_name);
CREATE INDEX IF NOT EXISTS idx_kill_players_kill ON kill_players (realm, kill_id, guild_id);
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
DROP TABLE IF EXISTS kill_players;
DROP TABLE IF EXISTS kill_history;
-- +goose StatementEnd
