package sqlite

import (
	"context"
	"database/sql"
	"time"

	"service/internal/domain/model"
)

type AuthRepoSQLite struct {
	DB *sql.DB
}

func (r *AuthRepoSQLite) GetUserByAccount(ctx context.Context, account string) (*model.User, string, error) {
	query := `
SELECT
	u.id,
	COALESCE(u.tenant_id, ''),
	COALESCE(t.display_name, ''),
	u.name,
	COALESCE(u.phone, ''),
	u.role,
	u.scope_type,
	u.token_version,
	u.must_change_password,
	COALESCE(u.password_updated_at, ''),
	c.password_hash
FROM user_credentials c
JOIN users u ON u.id = c.user_id
LEFT JOIN tenants t ON t.id = u.tenant_id
WHERE c.account = ?`
	args := []any{account}
	if tenantID := tenantScopeID(ctx); tenantID != "" {
		query += " AND COALESCE(u.tenant_id, '') = ?"
		args = append(args, tenantID)
	}
	row := r.DB.QueryRowContext(ctx, query, args...)
	return scanSQLiteUserWithHash(row)
}

func (r *AuthRepoSQLite) GetTokenVersionByUserID(ctx context.Context, userID string) (int, error) {
	query := `SELECT token_version FROM users WHERE id = ?`
	args := []any{userID}
	if tenantID := tenantScopeID(ctx); tenantID != "" {
		query += " AND COALESCE(tenant_id, '') = ?"
		args = append(args, tenantID)
	}
	var version int
	if err := r.DB.QueryRowContext(ctx, query, args...).Scan(&version); err != nil {
		return 0, mapNotFound(err)
	}
	return version, nil
}

func (r *AuthRepoSQLite) GetUserByID(ctx context.Context, userID string) (*model.User, error) {
	query := `
SELECT
	u.id,
	COALESCE(u.tenant_id, ''),
	COALESCE(t.display_name, ''),
	u.name,
	COALESCE(u.phone, ''),
	u.role,
	u.scope_type,
	u.token_version,
	u.must_change_password,
	COALESCE(u.password_updated_at, '')
FROM users u
LEFT JOIN tenants t ON t.id = u.tenant_id
WHERE u.id = ?`
	args := []any{userID}
	if tenantID := tenantScopeID(ctx); tenantID != "" {
		query += " AND COALESCE(u.tenant_id, '') = ?"
		args = append(args, tenantID)
	}
	row := r.DB.QueryRowContext(ctx, query, args...)
	user, err := scanSQLiteUser(row)
	if err != nil {
		return nil, err
	}
	return user, nil
}

func (r *AuthRepoSQLite) GetPasswordHashByUserID(ctx context.Context, userID string) (string, error) {
	query := `
SELECT c.password_hash
FROM user_credentials c
JOIN users u ON u.id = c.user_id
WHERE c.user_id = ?`
	args := []any{userID}
	if tenantID := tenantScopeID(ctx); tenantID != "" {
		query += " AND COALESCE(u.tenant_id, '') = ?"
		args = append(args, tenantID)
	}
	var hash string
	if err := r.DB.QueryRowContext(ctx, query, args...).Scan(&hash); err != nil {
		return "", mapNotFound(err)
	}
	return hash, nil
}

func (r *AuthRepoSQLite) UpdatePasswordByUserID(ctx context.Context, userID, passwordHash string, mustChange bool) error {
	tx, err := r.DB.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer tx.Rollback()

	if _, err := tx.ExecContext(ctx, `UPDATE user_credentials SET password_hash = ? WHERE user_id = ?`, passwordHash, userID); err != nil {
		return err
	}
	query := `UPDATE users SET must_change_password = ?, password_updated_at = ?, token_version = token_version + 1 WHERE id = ?`
	args := []any{boolToInt(mustChange), formatSQLiteTime(nowUTC()), userID}
	if tenantID := tenantScopeID(ctx); tenantID != "" {
		query += ` AND COALESCE(tenant_id, '') = ?`
		args = append(args, tenantID)
	}
	res, err := tx.ExecContext(ctx, query, args...)
	if err != nil {
		return err
	}
	if n, _ := res.RowsAffected(); n == 0 {
		return sql.ErrNoRows
	}
	return tx.Commit()
}

func (r *AuthRepoSQLite) CreateLoginChallenge(ctx context.Context, challenge *model.LoginChallenge) error {
	_, err := r.DB.ExecContext(ctx, `
INSERT INTO login_challenges (id, user_id, otp_hash, expires_at, attempts, verified_at, created_at)
VALUES (?, ?, ?, ?, ?, ?, ?)`,
		challenge.ID,
		challenge.UserID,
		challenge.OTPHash,
		formatSQLiteTime(challenge.ExpiresAt),
		challenge.Attempts,
		nullableSQLiteTime(challenge.VerifiedAt),
		formatSQLiteTime(challenge.CreatedAt),
	)
	return err
}

func (r *AuthRepoSQLite) GetLoginChallengeByID(ctx context.Context, challengeID string) (*model.LoginChallenge, error) {
	row := r.DB.QueryRowContext(ctx, `
SELECT id, user_id, otp_hash, expires_at, attempts, COALESCE(verified_at, ''), created_at
FROM login_challenges
WHERE id = ?`, challengeID)

	var (
		ch         model.LoginChallenge
		expiresAt  string
		verifiedAt sql.NullString
		createdAt  string
	)
	if err := row.Scan(&ch.ID, &ch.UserID, &ch.OTPHash, &expiresAt, &ch.Attempts, &verifiedAt, &createdAt); err != nil {
		return nil, mapNotFound(err)
	}
	var err error
	ch.ExpiresAt, err = parseSQLiteTime(expiresAt)
	if err != nil {
		return nil, err
	}
	ch.VerifiedAt, err = parseNullableSQLiteTime(verifiedAt)
	if err != nil {
		return nil, err
	}
	ch.CreatedAt, err = parseSQLiteTime(createdAt)
	if err != nil {
		return nil, err
	}
	return &ch, nil
}

func (r *AuthRepoSQLite) IncreaseLoginChallengeAttempts(ctx context.Context, challengeID string) error {
	_, err := r.DB.ExecContext(ctx, `UPDATE login_challenges SET attempts = attempts + 1 WHERE id = ?`, challengeID)
	return err
}

func (r *AuthRepoSQLite) MarkLoginChallengeVerified(ctx context.Context, challengeID string) error {
	_, err := r.DB.ExecContext(ctx, `UPDATE login_challenges SET verified_at = ? WHERE id = ?`, formatSQLiteTime(nowUTC()), challengeID)
	return err
}

func (r *AuthRepoSQLite) PhoneExists(ctx context.Context, phone string) (bool, error) {
	var exists int
	if err := r.DB.QueryRowContext(ctx, `SELECT EXISTS(SELECT 1 FROM users WHERE phone = ?)`, phone).Scan(&exists); err != nil {
		return false, err
	}
	return exists != 0, nil
}

func (r *AuthRepoSQLite) UpdatePhoneByUserID(ctx context.Context, userID, phone string) error {
	query := `UPDATE users SET phone = ? WHERE id = ?`
	args := []any{nullableString(phone), userID}
	if tenantID := tenantScopeID(ctx); tenantID != "" {
		query += ` AND COALESCE(tenant_id, '') = ?`
		args = append(args, tenantID)
	}
	_, err := r.DB.ExecContext(ctx, query, args...)
	return err
}

func (r *AuthRepoSQLite) UpdateDisplayNameByUserID(ctx context.Context, userID, name string) error {
	query := `UPDATE users SET name = ? WHERE id = ?`
	args := []any{name, userID}
	if tenantID := tenantScopeID(ctx); tenantID != "" {
		query += ` AND COALESCE(tenant_id, '') = ?`
		args = append(args, tenantID)
	}
	_, err := r.DB.ExecContext(ctx, query, args...)
	return err
}

func (r *AuthRepoSQLite) SavePhoneChangeChallenge(ctx context.Context, challenge *model.PhoneChangeChallenge) error {
	tx, err := r.DB.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer tx.Rollback()

	if _, err := tx.ExecContext(ctx, `DELETE FROM phone_change_challenges WHERE user_id = ? AND verified_at IS NULL`, challenge.UserID); err != nil {
		return err
	}
	if _, err := tx.ExecContext(ctx, `
INSERT INTO phone_change_challenges (
	id, user_id, new_phone, otp_hash, expires_at, attempts, resend_count, last_sent_at, verified_at, created_at
) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`,
		challenge.ID,
		challenge.UserID,
		challenge.NewPhone,
		challenge.OTPHash,
		formatSQLiteTime(challenge.ExpiresAt),
		challenge.Attempts,
		challenge.ResendCount,
		formatSQLiteTime(challenge.LastSentAt),
		nullableSQLiteTime(challenge.VerifiedAt),
		formatSQLiteTime(challenge.CreatedAt),
	); err != nil {
		return err
	}
	return tx.Commit()
}

func (r *AuthRepoSQLite) GetPhoneChangeChallengeByID(ctx context.Context, challengeID string) (*model.PhoneChangeChallenge, error) {
	row := r.DB.QueryRowContext(ctx, `
SELECT id, user_id, new_phone, otp_hash, expires_at, attempts, resend_count, last_sent_at, COALESCE(verified_at, ''), created_at
FROM phone_change_challenges
WHERE id = ?`, challengeID)
	var (
		ch         model.PhoneChangeChallenge
		expiresAt  string
		lastSentAt string
		verifiedAt sql.NullString
		createdAt  string
	)
	if err := row.Scan(&ch.ID, &ch.UserID, &ch.NewPhone, &ch.OTPHash, &expiresAt, &ch.Attempts, &ch.ResendCount, &lastSentAt, &verifiedAt, &createdAt); err != nil {
		return nil, mapNotFound(err)
	}
	var err error
	ch.ExpiresAt, err = parseSQLiteTime(expiresAt)
	if err != nil {
		return nil, err
	}
	ch.LastSentAt, err = parseSQLiteTime(lastSentAt)
	if err != nil {
		return nil, err
	}
	ch.VerifiedAt, err = parseNullableSQLiteTime(verifiedAt)
	if err != nil {
		return nil, err
	}
	ch.CreatedAt, err = parseSQLiteTime(createdAt)
	if err != nil {
		return nil, err
	}
	return &ch, nil
}

func (r *AuthRepoSQLite) IncreasePhoneChangeChallengeAttempts(ctx context.Context, challengeID string) error {
	_, err := r.DB.ExecContext(ctx, `UPDATE phone_change_challenges SET attempts = attempts + 1 WHERE id = ?`, challengeID)
	return err
}

func (r *AuthRepoSQLite) MarkPhoneChangeChallengeVerified(ctx context.Context, challengeID string) error {
	_, err := r.DB.ExecContext(ctx, `UPDATE phone_change_challenges SET verified_at = ? WHERE id = ?`, formatSQLiteTime(nowUTC()), challengeID)
	return err
}

type sqliteUserScanner interface {
	Scan(dest ...any) error
}

func scanSQLiteUser(row sqliteUserScanner) (*model.User, error) {
	var (
		u                 model.User
		passwordUpdatedAt sql.NullString
		mustChange        int
	)
	if err := row.Scan(
		&u.ID,
		&u.TenantID,
		&u.TenantName,
		&u.Name,
		&u.Phone,
		&u.Role,
		&u.ScopeType,
		&u.TokenVersion,
		&mustChange,
		&passwordUpdatedAt,
	); err != nil {
		return nil, mapNotFound(err)
	}
	u.MustChangePassword = intToBool(mustChange)
	parsed, err := parseNullableSQLiteTime(passwordUpdatedAt)
	if err != nil {
		return nil, err
	}
	u.PasswordUpdatedAt = parsed
	return &u, nil
}

func scanSQLiteUserWithHash(row sqliteUserScanner) (*model.User, string, error) {
	var hash string
	var (
		u                 model.User
		passwordUpdatedAt sql.NullString
		mustChange        int
	)
	if err := row.Scan(
		&u.ID,
		&u.TenantID,
		&u.TenantName,
		&u.Name,
		&u.Phone,
		&u.Role,
		&u.ScopeType,
		&u.TokenVersion,
		&mustChange,
		&passwordUpdatedAt,
		&hash,
	); err != nil {
		return nil, "", mapNotFound(err)
	}
	u.MustChangePassword = intToBool(mustChange)
	parsed, err := parseNullableSQLiteTime(passwordUpdatedAt)
	if err != nil {
		return nil, "", err
	}
	u.PasswordUpdatedAt = parsed
	return &u, hash, nil
}

func nowUTC() time.Time {
	return time.Now().UTC()
}
