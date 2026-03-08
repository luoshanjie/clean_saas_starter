-- Demo baseline schema init for SQLite SaaS kernel
-- 用于在全新 SQLite 数据库一次性初始化“当前最终结构”
-- This schema intentionally does not include PostgreSQL RLS or policy DDL.

PRAGMA foreign_keys = ON;

BEGIN;

CREATE TABLE IF NOT EXISTS tenants (
    id            TEXT PRIMARY KEY,
    name          TEXT NOT NULL,
    region_code   TEXT,
    disabled      INTEGER NOT NULL DEFAULT 0 CHECK (disabled IN (0, 1)),
    display_name  TEXT NOT NULL,
    province      TEXT,
    city          TEXT,
    district      TEXT,
    address       TEXT,
    contact_name  TEXT,
    contact_phone TEXT,
    remark        TEXT,
    status        TEXT NOT NULL DEFAULT 'active',
    created_at    TEXT NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at    TEXT NOT NULL DEFAULT CURRENT_TIMESTAMP,
    CONSTRAINT tenants_status_check CHECK (status IN ('active', 'inactive'))
);
CREATE UNIQUE INDEX IF NOT EXISTS uq_tenants_display_name ON tenants (display_name);
CREATE INDEX IF NOT EXISTS idx_tenants_status_created_id ON tenants (status, created_at DESC, id DESC);
CREATE INDEX IF NOT EXISTS idx_tenants_region_created_id ON tenants (province, city, district, created_at DESC, id DESC);

CREATE TABLE IF NOT EXISTS users (
    id                   TEXT PRIMARY KEY,
    tenant_id            TEXT NULL REFERENCES tenants(id) ON DELETE SET NULL,
    name                 TEXT NOT NULL,
    phone                TEXT,
    role                 TEXT NOT NULL,
    scope_type           TEXT NOT NULL,
    status               TEXT NOT NULL DEFAULT 'active',
    token_version        INTEGER NOT NULL DEFAULT 0,
    must_change_password INTEGER NOT NULL DEFAULT 0 CHECK (must_change_password IN (0, 1)),
    password_updated_at  TEXT,
    created_at           TEXT NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at           TEXT NOT NULL DEFAULT CURRENT_TIMESTAMP
);
CREATE INDEX IF NOT EXISTS idx_users_tenant ON users (tenant_id);
CREATE INDEX IF NOT EXISTS idx_users_role ON users (role);
CREATE UNIQUE INDEX IF NOT EXISTS uq_users_phone_not_null ON users (phone) WHERE phone IS NOT NULL;

CREATE TABLE IF NOT EXISTS user_credentials (
    user_id       TEXT PRIMARY KEY REFERENCES users(id) ON DELETE CASCADE,
    account       TEXT NOT NULL UNIQUE,
    password_hash TEXT NOT NULL,
    created_at    TEXT NOT NULL DEFAULT CURRENT_TIMESTAMP
);
CREATE UNIQUE INDEX IF NOT EXISTS uq_user_credentials_account ON user_credentials (account);

CREATE TABLE IF NOT EXISTS rbac_policies (
    id        INTEGER PRIMARY KEY AUTOINCREMENT,
    tenant_id TEXT NULL,
    ptype     TEXT NOT NULL,
    v0        TEXT,
    v1        TEXT,
    v2        TEXT,
    v3        TEXT,
    v4        TEXT,
    v5        TEXT
);
CREATE INDEX IF NOT EXISTS idx_rbac_tenant ON rbac_policies (tenant_id);

CREATE TABLE IF NOT EXISTS files (
    id         TEXT PRIMARY KEY,
    tenant_id  TEXT NULL REFERENCES tenants(id) ON DELETE SET NULL,
    bucket     TEXT NOT NULL,
    object_key TEXT NOT NULL,
    size       INTEGER NOT NULL DEFAULT 0,
    mime       TEXT NOT NULL,
    owner_type TEXT NOT NULL,
    owner_id   TEXT NOT NULL,
    created_at TEXT NOT NULL DEFAULT CURRENT_TIMESTAMP
);
CREATE INDEX IF NOT EXISTS idx_files_tenant ON files (tenant_id);
CREATE INDEX IF NOT EXISTS idx_files_owner ON files (owner_type, owner_id);

CREATE TABLE IF NOT EXISTS login_challenges (
    id          TEXT PRIMARY KEY,
    user_id     TEXT NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    otp_hash    TEXT NOT NULL,
    expires_at  TEXT NOT NULL,
    attempts    INTEGER NOT NULL DEFAULT 0,
    verified_at TEXT NULL,
    created_at  TEXT NOT NULL DEFAULT CURRENT_TIMESTAMP,
    CHECK (attempts >= 0)
);
CREATE INDEX IF NOT EXISTS idx_login_challenges_user ON login_challenges (user_id);
CREATE INDEX IF NOT EXISTS idx_login_challenges_expires ON login_challenges (expires_at);

CREATE TABLE IF NOT EXISTS audit_logs (
    id                    TEXT PRIMARY KEY,
    request_id            TEXT NOT NULL,
    operator_user_id      TEXT NULL,
    operator_role         TEXT NULL,
    operator_tenant_id    TEXT NULL,
    operator_username     TEXT NOT NULL DEFAULT '',
    operator_display_name TEXT NOT NULL DEFAULT '',
    target_type           TEXT NOT NULL,
    target_id             TEXT NULL,
    target_name           TEXT NULL,
    action                TEXT NOT NULL,
    module                TEXT NOT NULL,
    result                TEXT NOT NULL,
    error_code            TEXT NULL,
    before_json           TEXT NULL,
    after_json            TEXT NULL,
    changed_fields        TEXT NOT NULL DEFAULT '[]',
    ip                    TEXT NULL,
    user_agent            TEXT NULL,
    created_at            TEXT NOT NULL DEFAULT CURRENT_TIMESTAMP,
    CHECK (result IN ('success', 'fail'))
);
CREATE INDEX IF NOT EXISTS idx_audit_logs_created_at ON audit_logs (created_at DESC);
CREATE INDEX IF NOT EXISTS idx_audit_logs_request_id ON audit_logs (request_id);
CREATE INDEX IF NOT EXISTS idx_audit_logs_operator_user ON audit_logs (operator_user_id, created_at DESC);
CREATE INDEX IF NOT EXISTS idx_audit_logs_module_action ON audit_logs (module, action, created_at DESC);
CREATE INDEX IF NOT EXISTS idx_audit_logs_target ON audit_logs (target_type, target_id, created_at DESC);
CREATE INDEX IF NOT EXISTS idx_audit_logs_result ON audit_logs (result, created_at DESC);
CREATE INDEX IF NOT EXISTS idx_audit_logs_operator_tenant ON audit_logs (operator_tenant_id, created_at DESC);

CREATE TABLE IF NOT EXISTS file_upload_sessions (
    id           TEXT PRIMARY KEY,
    tenant_id    TEXT NULL REFERENCES tenants(id) ON DELETE SET NULL,
    uploaded_by  TEXT NOT NULL REFERENCES users(id) ON DELETE RESTRICT,
    scope_type   TEXT NOT NULL DEFAULT '',
    biz_type     TEXT NOT NULL,
    file_name    TEXT NOT NULL,
    content_type TEXT NOT NULL DEFAULT '',
    size_bytes   INTEGER NOT NULL DEFAULT 0,
    file_url     TEXT NOT NULL,
    status       TEXT NOT NULL,
    expires_at   TEXT NOT NULL,
    confirmed_at TEXT NULL,
    mime_type    TEXT NOT NULL DEFAULT '',
    duration_sec INTEGER NOT NULL DEFAULT 0,
    deleted_at   TEXT NULL,
    last_error   TEXT NOT NULL DEFAULT '',
    created_at   TEXT NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at   TEXT NOT NULL DEFAULT CURRENT_TIMESTAMP,
    CONSTRAINT chk_file_upload_status CHECK (status IN ('pending_confirm', 'confirmed', 'cleaned')),
    CONSTRAINT chk_file_upload_duration_non_negative CHECK (duration_sec >= 0)
);
CREATE INDEX IF NOT EXISTS idx_file_upload_sessions_status_expires_at
    ON file_upload_sessions (status, expires_at);
CREATE INDEX IF NOT EXISTS idx_file_upload_sessions_uploaded_by_created_at
    ON file_upload_sessions (uploaded_by, created_at DESC);
CREATE INDEX IF NOT EXISTS idx_file_upload_sessions_available_for_bind
    ON file_upload_sessions (status, deleted_at, expires_at, mime_type, duration_sec);

CREATE TABLE IF NOT EXISTS phone_change_challenges (
    id           TEXT PRIMARY KEY,
    user_id      TEXT NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    new_phone    TEXT NOT NULL,
    otp_hash     TEXT NOT NULL,
    expires_at   TEXT NOT NULL,
    attempts     INTEGER NOT NULL DEFAULT 0,
    resend_count INTEGER NOT NULL DEFAULT 0,
    last_sent_at TEXT NOT NULL DEFAULT CURRENT_TIMESTAMP,
    verified_at  TEXT NULL,
    created_at   TEXT NOT NULL DEFAULT CURRENT_TIMESTAMP,
    CHECK (attempts >= 0),
    CHECK (resend_count >= 0)
);
CREATE INDEX IF NOT EXISTS idx_phone_change_challenges_user_id ON phone_change_challenges (user_id);
CREATE INDEX IF NOT EXISTS idx_phone_change_challenges_expires_at ON phone_change_challenges (expires_at);
CREATE UNIQUE INDEX IF NOT EXISTS uq_phone_change_challenges_user_unverified
    ON phone_change_challenges (user_id)
    WHERE verified_at IS NULL;

COMMIT;
