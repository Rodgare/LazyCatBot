-- +goose Up
-- +goose StatementBegin
CREATE TABLE IF NOT EXISTS mythic_leaderboard (
    realm TEXT NOT NULL,
    guid INTEGER NOT NULL,
    name TEXT NOT NULL,
    class_id INTEGER NOT NULL,
    spec_id INTEGER NOT NULL,
    current_score REAL NOT NULL,
    position INTEGER NOT NULL,
    total_runs INTEGER NOT NULL,
    timed_runs INTEGER NOT NULL,
    best_key INTEGER NOT NULL,
    zodiac INTEGER NOT NULL,
    season INTEGER NOT NULL,
    week_id INTEGER NOT NULL,
    updated_at INTEGER NOT NULL,
    PRIMARY KEY (realm, guid)
);
CREATE INDEX IF NOT EXISTS idx_mythic_lb_spec ON mythic_leaderboard (realm, class_id, spec_id, current_score DESC);
CREATE INDEX IF NOT EXISTS idx_mythic_lb_name ON mythic_leaderboard (realm, name);
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
DROP TABLE IF EXISTS mythic_leaderboard;
-- +goose StatementEnd