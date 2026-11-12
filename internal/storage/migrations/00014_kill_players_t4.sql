-- +goose Up
-- +goose StatementBegin
ALTER TABLE kill_players ADD COLUMN t4 INTEGER NOT NULL DEFAULT 0;
ALTER TABLE kill_players ADD COLUMN set_pieces TEXT NOT NULL DEFAULT '';
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
ALTER TABLE kill_players DROP COLUMN set_pieces;
ALTER TABLE kill_players DROP COLUMN t4;
-- +goose StatementEnd
