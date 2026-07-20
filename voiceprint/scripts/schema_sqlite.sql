-- SQLite DDL for voiceprints (LingEchoX AutoMigrate also creates this).
-- Use when sharing ./lingecho.db with the Go server.

CREATE TABLE IF NOT EXISTS voiceprints (
    id INTEGER NOT NULL PRIMARY KEY,
    tenant_id INTEGER NOT NULL,
    assistant_id INTEGER DEFAULT NULL,
    subject_id INTEGER DEFAULT NULL,
    scene TEXT NOT NULL DEFAULT 'business',
    name TEXT NOT NULL,
    provider TEXT NOT NULL DEFAULT 'http',
    feature_id TEXT NOT NULL,
    status TEXT NOT NULL DEFAULT 'active',
    description TEXT DEFAULT NULL,
    feature_vector BLOB DEFAULT NULL,
    created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
    updated_at DATETIME DEFAULT CURRENT_TIMESTAMP,
    deleted_at DATETIME DEFAULT NULL,
    create_by TEXT DEFAULT NULL,
    update_by TEXT DEFAULT NULL,
    remark TEXT DEFAULT NULL
);

CREATE UNIQUE INDEX IF NOT EXISTS uk_tenant_feature ON voiceprints (tenant_id, feature_id);
CREATE INDEX IF NOT EXISTS idx_voiceprints_tenant_id ON voiceprints (tenant_id);
CREATE INDEX IF NOT EXISTS idx_voiceprints_assistant_id ON voiceprints (assistant_id);
CREATE INDEX IF NOT EXISTS idx_voiceprints_scene ON voiceprints (scene);
CREATE INDEX IF NOT EXISTS idx_voiceprints_feature_id ON voiceprints (feature_id);
CREATE INDEX IF NOT EXISTS idx_voiceprints_deleted_at ON voiceprints (deleted_at);
