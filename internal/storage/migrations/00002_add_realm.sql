-- +goose Up
-- +goose StatementBegin
ALTER TABLE subscribe ADD COLUMN realm TEXT DEFAULT 'x3';
ALTER TABLE player_subscribe ADD COLUMN realm TEXT DEFAULT 'x3';
ALTER TABLE guild_members ADD COLUMN realm TEXT DEFAULT 'x3';
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
ALTER TABLE subscribe DROP COLUMN realm;
ALTER TABLE player_subscribe DROP COLUMN realm;
ALTER TABLE guild_members DROP COLUMN realm;
-- +goose StatementEnd
