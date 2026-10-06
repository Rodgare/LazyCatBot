-- +goose Up
-- +goose StatementBegin
CREATE TABLE IF NOT EXISTS leaderboard_new (
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
    realm TEXT DEFAULT 'x3',
    PRIMARY KEY (raid_id, boss_id, class_id, spec_id, player_name, realm)
);
CREATE INDEX IF NOT EXISTS idx_lb_new_rank ON leaderboard_new (raid_id, boss_id, class_id, spec_id, dps DESC);
CREATE INDEX IF NOT EXISTS idx_lb_new_hps ON leaderboard_new (raid_id, boss_id, class_id, spec_id, hps DESC);

INSERT OR IGNORE INTO leaderboard_new (raid_id, boss_id, class_id, spec_id, player_name, ilvl, guild_id, zodiac, category, t4, role, dps, hps, realm)
SELECT raid_id, boss_id, class_id, spec_id, player_name, ilvl, guild_id, zodiac, category, t4, role, dps, hps, COALESCE(realm, 'x3') FROM leaderboard;

DROP TABLE leaderboard;
ALTER TABLE leaderboard_new RENAME TO leaderboard;

ALTER TABLE processed_kills ADD COLUMN created_at INTEGER DEFAULT 0;

CREATE TABLE IF NOT EXISTS leaderboard_sync_state (
    realm TEXT NOT NULL,
    sync_date TEXT NOT NULL,
    raid_id INTEGER NOT NULL,
    boss_id INTEGER NOT NULL,
    class_id INTEGER NOT NULL,
    spec_id INTEGER NOT NULL,
    PRIMARY KEY (realm, sync_date, raid_id, boss_id, class_id, spec_id)
);
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
DROP TABLE IF EXISTS leaderboard_sync_state;

CREATE TABLE IF NOT EXISTS leaderboard_old (
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

INSERT OR IGNORE INTO leaderboard_old (raid_id, boss_id, class_id, spec_id, player_name, ilvl, guild_id, zodiac, category, t4, role, dps, hps)
SELECT raid_id, boss_id, class_id, spec_id, player_name, ilvl, guild_id, zodiac, category, t4, role, dps, hps FROM leaderboard;

DROP TABLE leaderboard;
ALTER TABLE leaderboard_old RENAME TO leaderboard;

ALTER TABLE processed_kills DROP COLUMN created_at;
-- +goose StatementEnd