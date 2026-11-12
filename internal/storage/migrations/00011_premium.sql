-- +goose Up
-- +goose StatementBegin
CREATE TABLE IF NOT EXISTS premium_entitlements (
    channel_id TEXT PRIMARY KEY,
    plan TEXT NOT NULL,
    source TEXT NOT NULL DEFAULT '',
    status TEXT NOT NULL DEFAULT 'active',
    expires_at INTEGER NOT NULL DEFAULT 0,
    created_at INTEGER NOT NULL,
    updated_at INTEGER NOT NULL
);
CREATE INDEX IF NOT EXISTS idx_premium_entitlements_expires ON premium_entitlements (expires_at);
CREATE INDEX IF NOT EXISTS idx_premium_entitlements_status ON premium_entitlements (status);

CREATE TABLE IF NOT EXISTS premium_codes (
    code TEXT PRIMARY KEY,
    channel_id TEXT NOT NULL,
    plan TEXT NOT NULL,
    created_by TEXT NOT NULL DEFAULT '',
    created_at INTEGER NOT NULL,
    expires_at INTEGER NOT NULL,
    used_at INTEGER NOT NULL DEFAULT 0,
    used_by TEXT NOT NULL DEFAULT ''
);
CREATE INDEX IF NOT EXISTS idx_premium_codes_channel ON premium_codes (channel_id);

CREATE TABLE IF NOT EXISTS premium_audit (
    id INTEGER PRIMARY KEY,
    channel_id TEXT NOT NULL,
    action TEXT NOT NULL,
    plan TEXT NOT NULL DEFAULT '',
    source TEXT NOT NULL DEFAULT '',
    actor TEXT NOT NULL DEFAULT '',
    details TEXT NOT NULL DEFAULT '',
    created_at INTEGER NOT NULL
);
CREATE INDEX IF NOT EXISTS idx_premium_audit_channel ON premium_audit (channel_id, created_at);
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
DROP TABLE IF EXISTS premium_audit;
DROP TABLE IF EXISTS premium_codes;
DROP TABLE IF EXISTS premium_entitlements;
-- +goose StatementEnd
