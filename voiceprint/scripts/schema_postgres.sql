-- PostgreSQL DDL for voiceprints (LingEchoX AutoMigrate also creates this).

CREATE TABLE IF NOT EXISTS voiceprints (
    id BIGINT PRIMARY KEY,
    tenant_id BIGINT NOT NULL,
    assistant_id BIGINT DEFAULT NULL,
    subject_id BIGINT DEFAULT NULL,
    scene VARCHAR(32) NOT NULL DEFAULT 'business',
    name VARCHAR(128) NOT NULL,
    provider VARCHAR(32) NOT NULL DEFAULT 'http',
    feature_id VARCHAR(128) NOT NULL,
    status VARCHAR(24) NOT NULL DEFAULT 'active',
    description VARCHAR(512) DEFAULT NULL,
    feature_vector BYTEA DEFAULT NULL,
    created_at TIMESTAMPTZ DEFAULT NOW(),
    updated_at TIMESTAMPTZ DEFAULT NOW(),
    deleted_at TIMESTAMPTZ DEFAULT NULL,
    create_by VARCHAR(128) DEFAULT NULL,
    update_by VARCHAR(128) DEFAULT NULL,
    remark VARCHAR(128) DEFAULT NULL
);

CREATE UNIQUE INDEX IF NOT EXISTS uk_tenant_feature ON voiceprints (tenant_id, feature_id);
CREATE INDEX IF NOT EXISTS idx_voiceprints_tenant_id ON voiceprints (tenant_id);
CREATE INDEX IF NOT EXISTS idx_voiceprints_assistant_id ON voiceprints (assistant_id);
CREATE INDEX IF NOT EXISTS idx_voiceprints_scene ON voiceprints (scene);
CREATE INDEX IF NOT EXISTS idx_voiceprints_feature_id ON voiceprints (feature_id);
CREATE INDEX IF NOT EXISTS idx_voiceprints_deleted_at ON voiceprints (deleted_at);
