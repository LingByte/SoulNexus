-- LingEchoX 主库声纹相关 DDL（MySQL）
-- SQLite / PostgreSQL 见 schema_sqlite.sql / schema_postgres.sql
-- GORM AutoMigrate 也会维护这些结构；本脚本用于手工建库或对照。

CREATE TABLE IF NOT EXISTS voiceprints (
    id BIGINT UNSIGNED NOT NULL PRIMARY KEY,
    tenant_id BIGINT UNSIGNED NOT NULL,
    assistant_id BIGINT UNSIGNED DEFAULT NULL COMMENT '业务声纹可选绑定助手；一助手可绑多条',
    subject_id BIGINT UNSIGNED DEFAULT NULL COMMENT '逻辑说话人主体',
    scene VARCHAR(32) NOT NULL DEFAULT 'business' COMMENT 'business|account',
    name VARCHAR(128) NOT NULL,
    provider VARCHAR(32) NOT NULL DEFAULT 'http',
    feature_id VARCHAR(128) NOT NULL,
    status VARCHAR(24) NOT NULL DEFAULT 'active',
    description VARCHAR(512) DEFAULT NULL,
    feature_vector LONGBLOB DEFAULT NULL COMMENT 'HTTP 声纹服务写入的 embedding',
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
    deleted_at TIMESTAMP NULL DEFAULT NULL,
    create_by VARCHAR(128) DEFAULT NULL,
    update_by VARCHAR(128) DEFAULT NULL,
    remark VARCHAR(128) DEFAULT NULL,
    UNIQUE KEY uk_tenant_feature (tenant_id, feature_id),
    INDEX idx_tenant_id (tenant_id),
    INDEX idx_assistant_id (assistant_id),
    INDEX idx_subject_id (subject_id),
    INDEX idx_scene (scene),
    INDEX idx_provider (provider),
    INDEX idx_status (status),
    INDEX idx_feature_id (feature_id),
    INDEX idx_tenant_assistant (tenant_id, assistant_id),
    INDEX idx_tenant_scene (tenant_id, scene),
    INDEX idx_voiceprints_deleted_at (deleted_at)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;

-- ALTER TABLE tenant_users
--     ADD COLUMN voiceprint_id BIGINT UNSIGNED DEFAULT NULL,
--     ADD INDEX idx_tenant_users_voiceprint_id (voiceprint_id);
