-- +goose Up
-- +goose StatementBegin

CREATE TABLE IF NOT EXISTS actual_raids (
    raid_id INTEGER,
    boss_id INTEGER,
    server_id INTEGER,
    PRIMARY KEY (raid_id, boss_id, server_id)
);

CREATE TABLE IF NOT EXISTS subscribe (
    guild_id INTEGER,
    channel_id TEXT,
    discord_id TEXT,
    is_send INTEGER DEFAULT 1,
    PRIMARY KEY (guild_id, channel_id, discord_id)
);
CREATE INDEX IF NOT EXISTS idx_rank_lookup ON subscribe (guild_id, channel_id, discord_id);
CREATE TABLE IF NOT EXISTS processed_kills (
    kill_id INTEGER,
    channel_id TEXT,
    PRIMARY KEY (kill_id, channel_id)
);
CREATE TABLE IF NOT EXISTS player_subscribe (
    id INTEGER,
    name TEXT,
    channel_id TEXT,
    discord_id TEXT,
    PRIMARY KEY (id, channel_id)
);
CREATE TABLE IF NOT EXISTS guild_members (
    id INTEGER,
    name TEXT,
    ilvl INTEGER,
    guild_id INTEGER,
    PRIMARY KEY (name, guild_id)
);
CREATE INDEX IF NOT EXISTS idx_gm_name ON guild_members (name);
CREATE TABLE IF NOT EXISTS leaderboard (
    raid_id INTEGER,
    boss_id INTEGER,
    class_id INTEGER,
    spec_id INTEGER,
    player_name TEXT,
    ilvl INTEGER,
    guild_id INTEGER,
    zodiac INTEGER,
    category INTEGER,
    t4 INTEGER,
    role TEXT,
    dps INTEGER,
    hps INTEGER,
    PRIMARY KEY (raid_id, boss_id, class_id, spec_id, player_name)
);
CREATE INDEX IF NOT EXISTS idx_leaderboard_rank ON leaderboard (raid_id, boss_id, class_id, spec_id, dps DESC);
CREATE INDEX IF NOT EXISTS idx_leaderboard_hps ON leaderboard (raid_id, boss_id, class_id, spec_id, hps DESC);
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
DROP TABLE IF EXISTS actual_raids;
DROP TABLE IF EXISTS subscribe;
DROP TABLE IF EXISTS processed_kills;
DROP TABLE IF EXISTS player_subscribe;
DROP TABLE IF EXISTS guild_members;
DROP TABLE IF EXISTS leaderboard;
-- +goose StatementEnd
