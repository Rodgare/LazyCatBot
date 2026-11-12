-- +goose Up
-- +goose StatementBegin
ALTER TABLE premium_entitlements ADD COLUMN guild_id TEXT NOT NULL DEFAULT '';
ALTER TABLE premium_codes ADD COLUMN guild_id TEXT NOT NULL DEFAULT '';
ALTER TABLE premium_audit ADD COLUMN guild_id TEXT NOT NULL DEFAULT '';
CREATE INDEX IF NOT EXISTS idx_premium_entitlements_guild ON premium_entitlements (guild_id);
CREATE INDEX IF NOT EXISTS idx_premium_codes_guild ON premium_codes (guild_id);
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
DROP INDEX IF EXISTS idx_premium_codes_guild;
DROP INDEX IF EXISTS idx_premium_entitlements_guild;
ALTER TABLE premium_audit DROP COLUMN guild_id;
ALTER TABLE premium_codes DROP COLUMN guild_id;
ALTER TABLE premium_entitlements DROP COLUMN guild_id;
-- +goose StatementEnd
