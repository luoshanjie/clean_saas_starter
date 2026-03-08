BEGIN;

CREATE TABLE IF NOT EXISTS tenants (
    id            uuid PRIMARY KEY,
    name          text NOT NULL,
    region_code   text,
    disabled      boolean NOT NULL DEFAULT false,
    display_name  text NOT NULL,
    province      text,
    city          text,
    district      text,
    address       text,
    contact_name  text,
    contact_phone text,
    remark        text,
    status        text NOT NULL DEFAULT 'active',
    created_at    timestamptz NOT NULL DEFAULT now(),
    updated_at    timestamptz NOT NULL DEFAULT now(),
    CONSTRAINT tenants_status_check CHECK (status IN ('active', 'inactive'))
);
CREATE UNIQUE INDEX IF NOT EXISTS uq_tenants_display_name ON tenants (display_name);
CREATE INDEX IF NOT EXISTS idx_tenants_status_created_id ON tenants (status, created_at DESC, id DESC);
CREATE INDEX IF NOT EXISTS idx_tenants_region_created_id ON tenants (province, city, district, created_at DESC, id DESC);

ALTER TABLE tenants ENABLE ROW LEVEL SECURITY;
DROP POLICY IF EXISTS tenants_platform_only ON tenants;
CREATE POLICY tenants_platform_only ON tenants
    USING (current_setting('app.scope_type', true) = 'platform');

CREATE TABLE IF NOT EXISTS users (
    id                   uuid PRIMARY KEY,
    tenant_id            uuid NULL REFERENCES tenants(id) ON DELETE SET NULL,
    name                 text NOT NULL,
    phone                text,
    role                 text NOT NULL,
    scope_type           text NOT NULL,
    status               text NOT NULL DEFAULT 'active',
    token_version        integer NOT NULL DEFAULT 0,
    must_change_password boolean NOT NULL DEFAULT false,
    password_updated_at  timestamptz,
    created_at           timestamptz NOT NULL DEFAULT now(),
    updated_at           timestamptz NOT NULL DEFAULT now()
);
CREATE INDEX IF NOT EXISTS idx_users_tenant ON users (tenant_id);
CREATE INDEX IF NOT EXISTS idx_users_role ON users (role);
CREATE UNIQUE INDEX IF NOT EXISTS uq_users_phone_not_null ON users (phone) WHERE phone IS NOT NULL;

ALTER TABLE users ENABLE ROW LEVEL SECURITY;
DROP POLICY IF EXISTS users_tenant_isolation ON users;
CREATE POLICY users_tenant_isolation ON users
    USING (
        current_setting('app.scope_type', true) = 'platform'
        OR tenant_id::text = current_setting('app.tenant_id', true)
    );

CREATE TABLE IF NOT EXISTS user_credentials (
    user_id       uuid PRIMARY KEY REFERENCES users(id) ON DELETE CASCADE,
    account       text NOT NULL UNIQUE,
    password_hash text NOT NULL,
    created_at    timestamptz NOT NULL DEFAULT now()
);
CREATE UNIQUE INDEX IF NOT EXISTS uq_user_credentials_account ON user_credentials (account);

ALTER TABLE user_credentials ENABLE ROW LEVEL SECURITY;
DROP POLICY IF EXISTS user_credentials_tenant_isolation ON user_credentials;
CREATE POLICY user_credentials_tenant_isolation ON user_credentials
    USING (
        current_setting('app.scope_type', true) = 'platform'
        OR user_id IN (
            SELECT id
            FROM users
            WHERE tenant_id::text = current_setting('app.tenant_id', true)
        )
    );

CREATE TABLE IF NOT EXISTS rbac_policies (
    id        bigserial PRIMARY KEY,
    tenant_id uuid NULL,
    ptype     text NOT NULL,
    v0        text,
    v1        text,
    v2        text,
    v3        text,
    v4        text,
    v5        text
);
CREATE INDEX IF NOT EXISTS idx_rbac_tenant ON rbac_policies (tenant_id);

ALTER TABLE rbac_policies ENABLE ROW LEVEL SECURITY;
DROP POLICY IF EXISTS rbac_tenant_isolation ON rbac_policies;
CREATE POLICY rbac_tenant_isolation ON rbac_policies
    USING (
        current_setting('app.scope_type', true) = 'platform'
        OR tenant_id::text = current_setting('app.tenant_id', true)
    );

CREATE TABLE IF NOT EXISTS files (
    id         uuid PRIMARY KEY,
    tenant_id  uuid NOT NULL REFERENCES tenants(id) ON DELETE CASCADE,
    bucket     text NOT NULL,
    object_key text NOT NULL,
    size       bigint NOT NULL DEFAULT 0,
    mime       text NOT NULL,
    owner_type text NOT NULL,
    owner_id   uuid NOT NULL,
    created_at timestamptz NOT NULL DEFAULT now()
);
CREATE INDEX IF NOT EXISTS idx_files_tenant ON files (tenant_id);
CREATE INDEX IF NOT EXISTS idx_files_owner ON files (owner_type, owner_id);

ALTER TABLE files ENABLE ROW LEVEL SECURITY;
DROP POLICY IF EXISTS files_scope ON files;
CREATE POLICY files_scope ON files
    USING (
        current_setting('app.scope_type', true) = 'platform'
        OR tenant_id::text = current_setting('app.tenant_id', true)
    );

CREATE TABLE IF NOT EXISTS login_challenges (
    id          uuid PRIMARY KEY,
    user_id     uuid NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    otp_hash    text NOT NULL,
    expires_at  timestamptz NOT NULL,
    attempts    integer NOT NULL DEFAULT 0,
    verified_at timestamptz NULL,
    created_at  timestamptz NOT NULL DEFAULT now(),
    CHECK (attempts >= 0)
);
CREATE INDEX IF NOT EXISTS idx_login_challenges_user ON login_challenges (user_id);
CREATE INDEX IF NOT EXISTS idx_login_challenges_expires ON login_challenges (expires_at);

ALTER TABLE login_challenges ENABLE ROW LEVEL SECURITY;
DROP POLICY IF EXISTS login_challenges_owner_only ON login_challenges;
CREATE POLICY login_challenges_owner_only ON login_challenges
    USING (user_id::text = current_setting('app.user_id', true) OR current_setting('app.scope_type', true) = 'platform')
    WITH CHECK (user_id::text = current_setting('app.user_id', true) OR current_setting('app.scope_type', true) = 'platform');

CREATE TABLE IF NOT EXISTS audit_logs (
    id                    uuid PRIMARY KEY,
    request_id            text NOT NULL,
    operator_user_id      uuid NULL,
    operator_role         text NULL,
    operator_tenant_id    uuid NULL,
    operator_username     text NOT NULL DEFAULT '',
    operator_display_name text NOT NULL DEFAULT '',
    target_type           text NOT NULL,
    target_id             text NULL,
    target_name           text NULL,
    action                text NOT NULL,
    module                text NOT NULL,
    result                text NOT NULL,
    error_code            text NULL,
    before_json           jsonb NULL,
    after_json            jsonb NULL,
    changed_fields        jsonb NOT NULL DEFAULT '[]'::jsonb,
    ip                    text NULL,
    user_agent            text NULL,
    created_at            timestamptz NOT NULL DEFAULT now(),
    CHECK (result IN ('success', 'fail')),
    CHECK (jsonb_typeof(changed_fields) = 'array')
);
CREATE INDEX IF NOT EXISTS idx_audit_logs_created_at ON audit_logs (created_at DESC);
CREATE INDEX IF NOT EXISTS idx_audit_logs_request_id ON audit_logs (request_id);
CREATE INDEX IF NOT EXISTS idx_audit_logs_operator_user ON audit_logs (operator_user_id, created_at DESC);
CREATE INDEX IF NOT EXISTS idx_audit_logs_module_action ON audit_logs (module, action, created_at DESC);
CREATE INDEX IF NOT EXISTS idx_audit_logs_target ON audit_logs (target_type, target_id, created_at DESC);
CREATE INDEX IF NOT EXISTS idx_audit_logs_result ON audit_logs (result, created_at DESC);
CREATE INDEX IF NOT EXISTS idx_audit_logs_operator_tenant ON audit_logs (operator_tenant_id, created_at DESC);

CREATE TABLE IF NOT EXISTS file_upload_sessions (
    id           uuid PRIMARY KEY,
    tenant_id    uuid NULL REFERENCES tenants(id) ON DELETE SET NULL,
    uploaded_by  uuid NOT NULL REFERENCES users(id) ON DELETE RESTRICT,
    scope_type   text NOT NULL DEFAULT '',
    biz_type     text NOT NULL,
    file_name    text NOT NULL,
    content_type text NOT NULL DEFAULT '',
    size_bytes   bigint NOT NULL DEFAULT 0,
    file_url     text NOT NULL,
    status       text NOT NULL,
    expires_at   timestamptz NOT NULL,
    confirmed_at timestamptz NULL,
    mime_type    text NOT NULL DEFAULT '',
    duration_sec integer NOT NULL DEFAULT 0,
    deleted_at   timestamptz NULL,
    last_error   text NOT NULL DEFAULT '',
    created_at   timestamptz NOT NULL DEFAULT now(),
    updated_at   timestamptz NOT NULL DEFAULT now(),
    CONSTRAINT chk_file_upload_status CHECK (status IN ('pending_confirm', 'confirmed', 'cleaned')),
    CONSTRAINT chk_file_upload_duration_non_negative CHECK (duration_sec >= 0)
);
CREATE INDEX IF NOT EXISTS idx_file_upload_sessions_status_expires_at
    ON file_upload_sessions (status, expires_at);
CREATE INDEX IF NOT EXISTS idx_file_upload_sessions_uploaded_by_created_at
    ON file_upload_sessions (uploaded_by, created_at DESC);
CREATE INDEX IF NOT EXISTS idx_file_upload_sessions_available_for_bind
    ON file_upload_sessions (status, deleted_at, expires_at, mime_type, duration_sec);

ALTER TABLE file_upload_sessions ENABLE ROW LEVEL SECURITY;
DROP POLICY IF EXISTS file_upload_sessions_scope ON file_upload_sessions;
CREATE POLICY file_upload_sessions_scope ON file_upload_sessions
    USING (
        current_setting('app.scope_type', true) = 'platform'
        OR tenant_id::text = current_setting('app.tenant_id', true)
    );

CREATE TABLE IF NOT EXISTS phone_change_challenges (
    id           uuid PRIMARY KEY,
    user_id      uuid NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    new_phone    text NOT NULL,
    otp_hash     text NOT NULL,
    expires_at   timestamptz NOT NULL,
    attempts     integer NOT NULL DEFAULT 0,
    resend_count integer NOT NULL DEFAULT 0,
    last_sent_at timestamptz NOT NULL DEFAULT now(),
    verified_at  timestamptz NULL,
    created_at   timestamptz NOT NULL DEFAULT now(),
    CHECK (attempts >= 0),
    CHECK (resend_count >= 0),
    CHECK (new_phone ~ '^[1][3-9][0-9]{9}$')
);
CREATE INDEX IF NOT EXISTS idx_phone_change_challenges_user_id ON phone_change_challenges (user_id);
CREATE INDEX IF NOT EXISTS idx_phone_change_challenges_expires_at ON phone_change_challenges (expires_at);
CREATE UNIQUE INDEX IF NOT EXISTS uq_phone_change_challenges_user_unverified
    ON phone_change_challenges (user_id)
    WHERE verified_at IS NULL;

ALTER TABLE phone_change_challenges ENABLE ROW LEVEL SECURITY;
DROP POLICY IF EXISTS phone_change_challenges_owner_only ON phone_change_challenges;
CREATE POLICY phone_change_challenges_owner_only ON phone_change_challenges
    USING (user_id::text = current_setting('app.user_id', true))
    WITH CHECK (user_id::text = current_setting('app.user_id', true));

COMMIT;
