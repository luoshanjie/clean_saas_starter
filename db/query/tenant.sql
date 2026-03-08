-- name: CreateTenant :exec
INSERT INTO tenants (
    id, name, display_name, province, city, district, address,
    contact_name, contact_phone, remark, status, created_at, updated_at
)
VALUES (
    sqlc.arg(id),
    sqlc.arg(display_name),
    sqlc.arg(display_name),
    NULLIF(sqlc.arg(province), ''),
    NULLIF(sqlc.arg(city), ''),
    NULLIF(sqlc.arg(district), ''),
    NULLIF(sqlc.arg(address), ''),
    NULLIF(sqlc.arg(contact_name), ''),
    NULLIF(sqlc.arg(contact_phone), ''),
    NULLIF(sqlc.arg(remark), ''),
    sqlc.arg(status),
    sqlc.arg(created_at),
    sqlc.arg(updated_at)
);

-- name: CreateTenantAdminUser :exec
INSERT INTO users (id, tenant_id, name, phone, role, scope_type, status, created_at, must_change_password)
VALUES (
    sqlc.arg(user_id),
    sqlc.arg(tenant_id),
    sqlc.arg(name),
    NULLIF(sqlc.arg(phone), ''),
    'tenant_admin',
    'tenant',
    sqlc.arg(status),
    sqlc.arg(created_at),
    true
);

-- name: CreateTenantAdminCredential :exec
INSERT INTO user_credentials (user_id, account, password_hash, created_at)
VALUES (
    sqlc.arg(user_id),
    sqlc.arg(account),
    sqlc.arg(password_hash),
    sqlc.arg(created_at)
);

-- name: CountTenants :one
SELECT COUNT(1)
FROM tenants t
WHERE (sqlc.arg(keyword) = '' OR t.display_name ILIKE '%' || sqlc.arg(keyword) || '%')
  AND (sqlc.arg(province) = '' OR COALESCE(t.province, '') = sqlc.arg(province))
  AND (sqlc.arg(city) = '' OR COALESCE(t.city, '') = sqlc.arg(city))
  AND (sqlc.arg(district) = '' OR COALESCE(t.district, '') = sqlc.arg(district))
  AND (sqlc.arg(status) = '' OR t.status = sqlc.arg(status));

-- name: ListTenantsPage :many
SELECT
    t.id::text AS tenant_id,
    t.display_name,
    COALESCE(t.province, '') AS province,
    COALESCE(t.city, '') AS city,
    COALESCE(t.district, '') AS district,
    COALESCE(t.address, '') AS address,
    COALESCE(t.contact_name, '') AS contact_name,
    COALESCE(t.contact_phone, '') AS contact_phone,
    COALESCE(t.remark, '') AS remark,
    t.status,
    t.created_at,
    t.updated_at,
    COALESCE(ta.user_id::text, '') AS tenant_admin_user_id,
    COALESCE(ta.account, '') AS tenant_admin_account,
    COALESCE(ta.name, '') AS tenant_admin_name,
    COALESCE(ta.phone, '') AS tenant_admin_phone
FROM tenants t
LEFT JOIN LATERAL (
    SELECT u.id AS user_id, c.account, u.name, u.phone
    FROM users u
    LEFT JOIN user_credentials c ON c.user_id = u.id
    WHERE u.tenant_id = t.id AND u.role = 'tenant_admin'
    ORDER BY u.created_at ASC, u.id ASC
    LIMIT 1
) ta ON true
WHERE (sqlc.arg(keyword) = '' OR t.display_name ILIKE '%' || sqlc.arg(keyword) || '%')
  AND (sqlc.arg(province) = '' OR COALESCE(t.province, '') = sqlc.arg(province))
  AND (sqlc.arg(city) = '' OR COALESCE(t.city, '') = sqlc.arg(city))
  AND (sqlc.arg(district) = '' OR COALESCE(t.district, '') = sqlc.arg(district))
  AND (sqlc.arg(status) = '' OR t.status = sqlc.arg(status))
ORDER BY t.created_at DESC, t.id DESC
LIMIT sqlc.arg(limit_rows) OFFSET sqlc.arg(offset_rows);

-- name: GetTenantByID :one
SELECT
    id::text AS tenant_id,
    display_name,
    COALESCE(province, '') AS province,
    COALESCE(city, '') AS city,
    COALESCE(district, '') AS district,
    COALESCE(address, '') AS address,
    COALESCE(contact_name, '') AS contact_name,
    COALESCE(contact_phone, '') AS contact_phone,
    COALESCE(remark, '') AS remark,
    status,
    created_at,
    updated_at
FROM tenants
WHERE id = sqlc.arg(tenant_id);

-- name: UpdateTenant :execrows
UPDATE tenants
SET name = sqlc.arg(display_name),
    display_name = sqlc.arg(display_name),
    province = NULLIF(sqlc.arg(province), ''),
    city = NULLIF(sqlc.arg(city), ''),
    district = NULLIF(sqlc.arg(district), ''),
    address = NULLIF(sqlc.arg(address), ''),
    contact_name = NULLIF(sqlc.arg(contact_name), ''),
    contact_phone = NULLIF(sqlc.arg(contact_phone), ''),
    remark = NULLIF(sqlc.arg(remark), ''),
    updated_at = sqlc.arg(updated_at)
WHERE id = sqlc.arg(tenant_id);

-- name: ToggleTenantStatus :execrows
UPDATE tenants
SET status = sqlc.arg(status), updated_at = now()
WHERE id = sqlc.arg(tenant_id);

-- name: UpdateUsersStatusByTenantID :exec
UPDATE users
SET status = sqlc.arg(status)
WHERE tenant_id = sqlc.arg(tenant_id);

-- name: HasTenantAdmin :one
SELECT EXISTS (
    SELECT 1
    FROM users u
    JOIN user_credentials c ON c.user_id = u.id
    WHERE u.tenant_id = sqlc.arg(tenant_id)
      AND u.role = 'tenant_admin'
);

-- name: DisplayNameExists :one
SELECT EXISTS(SELECT 1 FROM tenants WHERE display_name = sqlc.arg(display_name));

-- name: TenantAdminAccountExists :one
SELECT EXISTS(SELECT 1 FROM user_credentials WHERE account = sqlc.arg(account));

-- name: TenantAdminPhoneExists :one
SELECT EXISTS(SELECT 1 FROM users WHERE phone = sqlc.arg(phone));

-- name: GetTenantAdminByTenantID :one
SELECT
    u.id::text AS user_id,
    COALESCE(u.tenant_id::text, '') AS tenant_id,
    u.name,
    COALESCE(u.phone, '') AS phone,
    u.role,
    u.scope_type,
    u.token_version,
    COALESCE(u.must_change_password, false) AS must_change_password,
    u.password_updated_at,
    c.account
FROM users u
JOIN user_credentials c ON c.user_id = u.id
WHERE u.tenant_id = sqlc.arg(tenant_id)
  AND u.role = 'tenant_admin'
  AND u.status = 'active'
ORDER BY u.created_at ASC, u.id ASC
LIMIT 1;

-- name: GetTenantAdminByUserID :one
SELECT
    u.id::text AS user_id,
    COALESCE(u.tenant_id::text, '') AS tenant_id,
    u.name,
    COALESCE(u.phone, '') AS phone,
    u.role,
    u.scope_type,
    u.token_version,
    COALESCE(u.must_change_password, false) AS must_change_password,
    u.password_updated_at,
    c.account
FROM users u
JOIN user_credentials c ON c.user_id = u.id
WHERE u.id = sqlc.arg(user_id)
  AND u.role = 'tenant_admin'
  AND u.status = 'active'
LIMIT 1;

-- name: UpdateTenantAdminProfile :exec
UPDATE users
SET name = sqlc.arg(name),
    phone = NULLIF(sqlc.arg(phone), '')
WHERE id = sqlc.arg(user_id);

-- name: UpdateTenantAdminAccount :exec
UPDATE user_credentials
SET account = sqlc.arg(account)
WHERE user_id = sqlc.arg(user_id);

-- name: UpdateTenantAdminPassword :exec
UPDATE user_credentials
SET password_hash = sqlc.arg(password_hash)
WHERE user_id = sqlc.arg(user_id);

-- name: MarkTenantAdminPasswordReset :exec
UPDATE users
SET must_change_password = true,
    password_updated_at = now(),
    token_version = token_version + 1
WHERE id = sqlc.arg(user_id);
