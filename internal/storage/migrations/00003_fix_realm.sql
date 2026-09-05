-- +goose Up
ALTER TABLE player_subscribe ADD COLUMN realm TEXT DEFAULT 'x3';
ALTER TABLE guild_members ADD COLUMN realm TEXT DEFAULT 'x3';
ALTER TABLE leaderboard ADD COLUMN realm TEXT DEFAULT 'x3';
ALTER TABLE actual_raids ADD COLUMN realm TEXT DEFAULT 'x3';

-- +goose Down
ALTER TABLE player_subscribe DROP COLUMN realm;
ALTER TABLE guild_members DROP COLUMN realm;
ALTER TABLE leaderboard DROP COLUMN realm;
ALTER TABLE actual_raids DROP COLUMN realm;
