-- name: AuditCreate :exec
INSERT INTO audit_logs (
    id, request_id, operator_user_id, operator_role, operator_tenant_id,
    operator_username, operator_display_name,
    target_type, target_id, target_name, action, module, result, error_code,
    before_json, after_json, changed_fields, ip, user_agent, created_at
)
VALUES (
    sqlc.arg(id)::uuid,
    sqlc.arg(request_id),
    NULLIF(sqlc.arg(operator_user_id), '')::uuid,
    NULLIF(sqlc.arg(operator_role), ''),
    NULLIF(sqlc.arg(operator_tenant_id), '')::uuid,
    COALESCE(NULLIF(sqlc.arg(operator_username), ''), (SELECT c.account FROM user_credentials c WHERE c.user_id = NULLIF(sqlc.arg(operator_user_id), '')::uuid LIMIT 1), ''),
    COALESCE(NULLIF(sqlc.arg(operator_display_name), ''), (SELECT u.name FROM users u WHERE u.id = NULLIF(sqlc.arg(operator_user_id), '')::uuid LIMIT 1), ''),
    sqlc.arg(target_type),
    NULLIF(sqlc.arg(target_id), ''),
    COALESCE(
        NULLIF(sqlc.arg(target_name), ''),
        CASE
            WHEN sqlc.arg(target_type) = 'user' THEN (SELECT c.account FROM user_credentials c WHERE c.user_id::text = NULLIF(sqlc.arg(target_id), '') LIMIT 1)
            ELSE ''
        END
    ),
    sqlc.arg(action),
    sqlc.arg(module),
    sqlc.arg(result),
    NULLIF(sqlc.arg(error_code), ''),
    sqlc.arg(before_json)::jsonb,
    sqlc.arg(after_json)::jsonb,
    sqlc.arg(changed_fields)::jsonb,
    NULLIF(sqlc.arg(ip), ''),
    NULLIF(sqlc.arg(user_agent), ''),
    sqlc.arg(created_at)
);

-- name: AuditCount :one
SELECT COUNT(1)
FROM audit_logs a
LEFT JOIN user_credentials c ON c.user_id::text = a.target_id
WHERE (sqlc.arg(module) = '' OR a.module = sqlc.arg(module))
  AND (sqlc.arg(action) = '' OR a.action = sqlc.arg(action))
  AND (sqlc.arg(result) = '' OR a.result = sqlc.arg(result))
  AND (sqlc.arg(operator_user_id) = '' OR a.operator_user_id::text = sqlc.arg(operator_user_id))
  AND (sqlc.arg(target_type) = '' OR a.target_type = sqlc.arg(target_type))
  AND (sqlc.arg(target_id) = '' OR a.target_id = sqlc.arg(target_id))
  AND (sqlc.arg(request_id) = '' OR a.request_id = sqlc.arg(request_id))
  AND (sqlc.arg(tenant_id) = '' OR a.operator_tenant_id::text = sqlc.arg(tenant_id))
  AND (sqlc.arg(date_from) = '' OR a.created_at >= sqlc.arg(date_from)::timestamptz)
  AND (sqlc.arg(date_to) = '' OR a.created_at <= sqlc.arg(date_to)::timestamptz)
  AND (sqlc.arg(keyword) = '' OR a.target_name ILIKE '%' || sqlc.arg(keyword) || '%' OR a.target_id ILIKE '%' || sqlc.arg(keyword) || '%');

-- name: AuditListPage :many
SELECT
    a.id::text AS id,
    a.request_id,
    COALESCE(a.operator_user_id::text, '') AS operator_user_id,
    COALESCE(a.operator_role, '') AS operator_role,
    COALESCE(a.operator_tenant_id::text, '') AS operator_tenant_id,
    COALESCE(a.operator_username, '') AS operator_username,
    COALESCE(a.operator_display_name, '') AS operator_display_name,
    a.target_type,
    COALESCE(a.target_id, '') AS target_id,
    COALESCE(NULLIF(a.target_name, ''), CASE WHEN a.target_type = 'user' THEN COALESCE(c.account, '') ELSE '' END) AS target_name,
    a.action,
    a.module,
    a.result,
    COALESCE(a.error_code, '') AS error_code,
    a.before_json,
    a.after_json,
    a.changed_fields,
    COALESCE(a.ip, '') AS ip,
    COALESCE(a.user_agent, '') AS user_agent,
    a.created_at
FROM audit_logs a
LEFT JOIN user_credentials c ON c.user_id::text = a.target_id
WHERE (sqlc.arg(module) = '' OR a.module = sqlc.arg(module))
  AND (sqlc.arg(action) = '' OR a.action = sqlc.arg(action))
  AND (sqlc.arg(result) = '' OR a.result = sqlc.arg(result))
  AND (sqlc.arg(operator_user_id) = '' OR a.operator_user_id::text = sqlc.arg(operator_user_id))
  AND (sqlc.arg(target_type) = '' OR a.target_type = sqlc.arg(target_type))
  AND (sqlc.arg(target_id) = '' OR a.target_id = sqlc.arg(target_id))
  AND (sqlc.arg(request_id) = '' OR a.request_id = sqlc.arg(request_id))
  AND (sqlc.arg(tenant_id) = '' OR a.operator_tenant_id::text = sqlc.arg(tenant_id))
  AND (sqlc.arg(date_from) = '' OR a.created_at >= sqlc.arg(date_from)::timestamptz)
  AND (sqlc.arg(date_to) = '' OR a.created_at <= sqlc.arg(date_to)::timestamptz)
  AND (sqlc.arg(keyword) = '' OR a.target_name ILIKE '%' || sqlc.arg(keyword) || '%' OR a.target_id ILIKE '%' || sqlc.arg(keyword) || '%')
ORDER BY a.created_at DESC, a.id DESC
LIMIT sqlc.arg(limit_rows) OFFSET sqlc.arg(offset_rows);

-- name: AuditGetByID :one
SELECT
    a.id::text AS id,
    a.request_id,
    COALESCE(a.operator_user_id::text, '') AS operator_user_id,
    COALESCE(a.operator_role, '') AS operator_role,
    COALESCE(a.operator_tenant_id::text, '') AS operator_tenant_id,
    COALESCE(a.operator_username, '') AS operator_username,
    COALESCE(a.operator_display_name, '') AS operator_display_name,
    a.target_type,
    COALESCE(a.target_id, '') AS target_id,
    COALESCE(NULLIF(a.target_name, ''), CASE WHEN a.target_type = 'user' THEN COALESCE(c.account, '') ELSE '' END) AS target_name,
    a.action,
    a.module,
    a.result,
    COALESCE(a.error_code, '') AS error_code,
    a.before_json,
    a.after_json,
    a.changed_fields,
    COALESCE(a.ip, '') AS ip,
    COALESCE(a.user_agent, '') AS user_agent,
    a.created_at
FROM audit_logs a
LEFT JOIN user_credentials c ON c.user_id::text = a.target_id
WHERE a.id = sqlc.arg(id)::uuid;
