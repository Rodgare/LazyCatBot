-- +goose Up
-- +goose StatementBegin
ALTER TABLE subscribe ADD COLUMN guild_name TEXT DEFAULT '';
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
ALTER TABLE subscribe DROP COLUMN guild_name;
-- +goose StatementEnd