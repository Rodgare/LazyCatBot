-- +goose Up
-- +goose StatementBegin
CREATE TABLE IF NOT EXISTS leaderboard_history (
    realm TEXT NOT NULL,
    raid_id INTEGER NOT NULL,
    boss_id INTEGER NOT NULL,
    class_id INTEGER NOT NULL,
    spec_id INTEGER NOT NULL,
    player_name TEXT NOT NULL,
    ilvl INTEGER NOT NULL DEFAULT 0,
    guild_id INTEGER NOT NULL DEFAULT 0,
    zodiac INTEGER NOT NULL DEFAULT 0,
    category INTEGER NOT NULL DEFAULT 0,
    t4 INTEGER NOT NULL DEFAULT 0,
    set_pieces TEXT NOT NULL DEFAULT '',
    role TEXT NOT NULL DEFAULT '',
    dps INTEGER NOT NULL DEFAULT 0,
    hps INTEGER NOT NULL DEFAULT 0,
    snapshot_week TEXT NOT NULL,
    captured_at INTEGER NOT NULL,
    PRIMARY KEY (realm, raid_id, boss_id, class_id, spec_id, player_name, snapshot_week)
);
CREATE INDEX IF NOT EXISTS idx_lb_history_boss ON leaderboard_history (realm, raid_id, boss_id, role, captured_at);
CREATE INDEX IF NOT EXISTS idx_lb_history_player ON leaderboard_history (realm, player_name);
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
DROP TABLE IF EXISTS leaderboard_history;
-- +goose StatementEnd
