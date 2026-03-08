-- name: AuthGetUserByAccount :one
SELECT
    u.id::text AS id,
    COALESCE(u.tenant_id::text, '') AS tenant_id,
    COALESCE(t.display_name, '') AS tenant_name,
    u.name,
    COALESCE(u.phone, '') AS phone,
    u.role,
    u.scope_type,
    u.token_version,
    COALESCE(u.must_change_password, false) AS must_change_password,
    u.password_updated_at,
    c.password_hash
FROM users u
JOIN user_credentials c ON c.user_id = u.id
LEFT JOIN tenants t ON t.id = u.tenant_id
WHERE c.account = sqlc.arg(account)
  AND u.status = 'active';

-- name: AuthGetTokenVersionByUserID :one
SELECT token_version
FROM users
WHERE id = sqlc.arg(user_id)::uuid
  AND status = 'active';

-- name: AuthGetUserByID :one
SELECT
    u.id::text AS id,
    COALESCE(u.tenant_id::text, '') AS tenant_id,
    COALESCE(t.display_name, '') AS tenant_name,
    u.name,
    COALESCE(u.phone, '') AS phone,
    u.role,
    u.scope_type,
    u.token_version,
    COALESCE(u.must_change_password, false) AS must_change_password,
    u.password_updated_at
FROM users u
LEFT JOIN tenants t ON t.id = u.tenant_id
WHERE u.id = sqlc.arg(user_id)::uuid
  AND u.status = 'active';

-- name: AuthGetPasswordHashByUserID :one
SELECT c.password_hash
FROM user_credentials c
JOIN users u ON u.id = c.user_id
WHERE c.user_id = sqlc.arg(user_id)::uuid
  AND u.status = 'active';

-- name: AuthUpdatePasswordHashByUserID :exec
UPDATE user_credentials
SET password_hash = sqlc.arg(password_hash)
WHERE user_id = sqlc.arg(user_id)::uuid;

-- name: AuthUpdatePasswordMetaByUserID :exec
UPDATE users
SET must_change_password = sqlc.arg(must_change_password),
    password_updated_at = now(),
    token_version = token_version + 1
WHERE id = sqlc.arg(user_id)::uuid;

-- name: AuthPhoneExists :one
SELECT EXISTS(
    SELECT 1 FROM users
    WHERE phone = sqlc.arg(phone)
      AND status = 'active'
);

-- name: AuthUpdatePhoneByUserID :exec
UPDATE users
SET phone = sqlc.arg(phone),
    token_version = token_version + 1
WHERE id = sqlc.arg(user_id)::uuid;

-- name: AuthUpdateDisplayNameByUserID :exec
UPDATE users
SET name = sqlc.arg(name)
WHERE id = sqlc.arg(user_id)::uuid
  AND status = 'active';

-- name: AuthCreateLoginChallenge :exec
INSERT INTO login_challenges (id, user_id, otp_hash, expires_at, attempts, verified_at, created_at)
VALUES (
    sqlc.arg(id)::uuid,
    sqlc.arg(user_id)::uuid,
    sqlc.arg(otp_hash),
    sqlc.arg(expires_at),
    sqlc.arg(attempts),
    sqlc.arg(verified_at),
    sqlc.arg(created_at)
);

-- name: AuthGetLoginChallengeByID :one
SELECT
    id::text AS id,
    user_id::text AS user_id,
    otp_hash,
    expires_at,
    attempts,
    verified_at,
    created_at
FROM login_challenges
WHERE id = sqlc.arg(challenge_id)::uuid;

-- name: AuthIncreaseLoginChallengeAttempts :exec
UPDATE login_challenges
SET attempts = attempts + 1
WHERE id = sqlc.arg(challenge_id)::uuid;

-- name: AuthMarkLoginChallengeVerified :exec
UPDATE login_challenges
SET verified_at = now()
WHERE id = sqlc.arg(challenge_id)::uuid
  AND verified_at IS NULL;

-- name: AuthDeleteUnverifiedPhoneChangeChallengesByUserID :exec
DELETE FROM phone_change_challenges
WHERE user_id = sqlc.arg(user_id)::uuid
  AND verified_at IS NULL;

-- name: AuthCreatePhoneChangeChallenge :exec
INSERT INTO phone_change_challenges (
    id, user_id, new_phone, otp_hash, expires_at, attempts, resend_count, last_sent_at, verified_at, created_at
) VALUES (
    sqlc.arg(id)::uuid,
    sqlc.arg(user_id)::uuid,
    sqlc.arg(new_phone),
    sqlc.arg(otp_hash),
    sqlc.arg(expires_at),
    sqlc.arg(attempts),
    sqlc.arg(resend_count),
    sqlc.arg(last_sent_at),
    sqlc.arg(verified_at),
    sqlc.arg(created_at)
);

-- name: AuthGetPhoneChangeChallengeByID :one
SELECT
    id::text AS id,
    user_id::text AS user_id,
    new_phone,
    otp_hash,
    expires_at,
    attempts,
    resend_count,
    last_sent_at,
    verified_at,
    created_at
FROM phone_change_challenges
WHERE id = sqlc.arg(challenge_id)::uuid;

-- name: AuthIncreasePhoneChangeChallengeAttempts :exec
UPDATE phone_change_challenges
SET attempts = attempts + 1
WHERE id = sqlc.arg(challenge_id)::uuid;

-- name: AuthMarkPhoneChangeChallengeVerified :exec
UPDATE phone_change_challenges
SET verified_at = now()
WHERE id = sqlc.arg(challenge_id)::uuid
  AND verified_at IS NULL;
