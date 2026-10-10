-- +goose Up
-- +goose StatementBegin
ALTER TABLE leaderboard ADD COLUMN set_pieces TEXT;
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
ALTER TABLE leaderboard DROP COLUMN set_pieces;
-- +goose StatementEnd