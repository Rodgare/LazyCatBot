-- +goose Up
-- +goose StatementBegin
ALTER TABLE subscribe ADD COLUMN is_myth_reps INTEGER DEFAULT 1;
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
ALTER TABLE subscribe DROP COLUMN is_myth_reps;
-- +goose StatementEnd
