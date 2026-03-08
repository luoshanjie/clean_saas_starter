package sqlite

import (
	"context"
	"database/sql"
	"strings"

	"service/internal/domain/model"
	"service/internal/domain/port"
)

type TenantRepoSQLite struct {
	DB *sql.DB
}

func (r *TenantRepoSQLite) CreateWithAdmin(ctx context.Context, in *model.TenantCreateInput) (*model.TenantCreateOutput, error) {
	tx, err := r.DB.BeginTx(ctx, nil)
	if err != nil {
		return nil, err
	}
	defer tx.Rollback()

	if _, err := tx.ExecContext(ctx, `
INSERT INTO tenants (
	id, name, display_name, province, city, district, address,
	contact_name, contact_phone, remark, status, created_at, updated_at
) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`,
		in.TenantID,
		in.DisplayName,
		in.DisplayName,
		nullableString(in.Province),
		nullableString(in.City),
		nullableString(in.District),
		nullableString(in.Address),
		nullableString(in.ContactName),
		nullableString(in.ContactPhone),
		nullableString(in.Remark),
		in.Status,
		formatSQLiteTime(in.CreatedAt),
		formatSQLiteTime(in.UpdatedAt),
	); err != nil {
		return nil, mapTenantConstraintError(err)
	}
	if _, err := tx.ExecContext(ctx, `
INSERT INTO users (
	id, tenant_id, name, phone, role, scope_type, status, token_version, must_change_password,
	password_updated_at, created_at, updated_at
) VALUES (?, ?, ?, ?, 'tenant_admin', 'tenant', ?, 0, 1, NULL, ?, ?)`,
		in.TenantAdminUserID,
		in.TenantID,
		in.TenantAdminName,
		nullableString(in.TenantAdminPhone),
		in.Status,
		formatSQLiteTime(in.TenantAdminCreatedAt),
		formatSQLiteTime(in.TenantAdminCreatedAt),
	); err != nil {
		return nil, mapTenantConstraintError(err)
	}
	if _, err := tx.ExecContext(ctx, `
INSERT INTO user_credentials (user_id, account, password_hash, created_at)
VALUES (?, ?, ?, ?)`,
		in.TenantAdminUserID,
		in.TenantAdminAccount,
		in.TenantAdminPassword,
		formatSQLiteTime(in.TenantAdminCreatedAt),
	); err != nil {
		return nil, mapTenantConstraintError(err)
	}
	if err := tx.Commit(); err != nil {
		return nil, err
	}
	return &model.TenantCreateOutput{
		TenantID:           in.TenantID,
		TenantAdminUserID:  in.TenantAdminUserID,
		TenantAdminAccount: in.TenantAdminAccount,
		TenantAdminName:    in.TenantAdminName,
		Status:             in.Status,
		CreatedAt:          in.CreatedAt,
	}, nil
}

func (r *TenantRepoSQLite) ListPage(ctx context.Context, filter port.TenantFilter) ([]*model.TenantListItem, int, error) {
	where, args := buildTenantFilter(filter)
	total := -1
	if filter.NeedTotal {
		query := `SELECT COUNT(1) FROM tenants t` + where
		if err := r.DB.QueryRowContext(ctx, query, args...).Scan(&total); err != nil {
			return nil, 0, err
		}
	}
	page, pageSize := normalizePage(filter.Page, filter.PageSize)
	offset := (page - 1) * pageSize
	args = append(args, pageSize, offset)
	query := `
SELECT
	t.id,
	t.display_name,
	COALESCE(t.province, ''),
	COALESCE(t.city, ''),
	COALESCE(t.district, ''),
	COALESCE(t.address, ''),
	COALESCE(t.contact_name, ''),
	COALESCE(t.contact_phone, ''),
	COALESCE(t.remark, ''),
	t.status,
	t.created_at,
	t.updated_at,
	COALESCE((
		SELECT u.id FROM users u
		WHERE u.tenant_id = t.id AND u.role = 'tenant_admin'
		ORDER BY u.created_at ASC, u.id ASC
		LIMIT 1
	), ''),
	COALESCE((
		SELECT c.account FROM users u
		JOIN user_credentials c ON c.user_id = u.id
		WHERE u.tenant_id = t.id AND u.role = 'tenant_admin'
		ORDER BY u.created_at ASC, u.id ASC
		LIMIT 1
	), ''),
	COALESCE((
		SELECT u.name FROM users u
		WHERE u.tenant_id = t.id AND u.role = 'tenant_admin'
		ORDER BY u.created_at ASC, u.id ASC
		LIMIT 1
	), ''),
	COALESCE((
		SELECT COALESCE(u.phone, '') FROM users u
		WHERE u.tenant_id = t.id AND u.role = 'tenant_admin'
		ORDER BY u.created_at ASC, u.id ASC
		LIMIT 1
	), '')
FROM tenants t` + where + `
ORDER BY t.created_at DESC, t.id DESC
LIMIT ? OFFSET ?`
	rows, err := r.DB.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, 0, err
	}
	defer rows.Close()
	out := []*model.TenantListItem{}
	for rows.Next() {
		var (
			item      model.TenantListItem
			tenant    model.Tenant
			createdAt string
			updatedAt string
		)
		if err := rows.Scan(
			&tenant.ID,
			&tenant.DisplayName,
			&tenant.Province,
			&tenant.City,
			&tenant.District,
			&tenant.Address,
			&tenant.ContactName,
			&tenant.ContactPhone,
			&tenant.Remark,
			&tenant.Status,
			&createdAt,
			&updatedAt,
			&item.TenantAdminUserID,
			&item.TenantAdminAccount,
			&item.TenantAdminName,
			&item.TenantAdminPhone,
		); err != nil {
			return nil, 0, err
		}
		var err error
		tenant.CreatedAt, err = parseSQLiteTime(createdAt)
		if err != nil {
			return nil, 0, err
		}
		tenant.UpdatedAt, err = parseSQLiteTime(updatedAt)
		if err != nil {
			return nil, 0, err
		}
		item.Tenant = &tenant
		out = append(out, &item)
	}
	return out, total, rows.Err()
}

func (r *TenantRepoSQLite) GetByID(ctx context.Context, tenantID string) (*model.Tenant, error) {
	row := r.DB.QueryRowContext(ctx, `
SELECT id, display_name, COALESCE(province, ''), COALESCE(city, ''), COALESCE(district, ''),
       COALESCE(address, ''), COALESCE(contact_name, ''), COALESCE(contact_phone, ''),
       COALESCE(remark, ''), status, created_at, updated_at
FROM tenants
WHERE id = ?`, tenantID)
	var (
		tenant               model.Tenant
		createdAt, updatedAt string
	)
	if err := row.Scan(
		&tenant.ID,
		&tenant.DisplayName,
		&tenant.Province,
		&tenant.City,
		&tenant.District,
		&tenant.Address,
		&tenant.ContactName,
		&tenant.ContactPhone,
		&tenant.Remark,
		&tenant.Status,
		&createdAt,
		&updatedAt,
	); err != nil {
		return nil, mapNotFound(err)
	}
	var err error
	tenant.CreatedAt, err = parseSQLiteTime(createdAt)
	if err != nil {
		return nil, err
	}
	tenant.UpdatedAt, err = parseSQLiteTime(updatedAt)
	if err != nil {
		return nil, err
	}
	return &tenant, nil
}

func (r *TenantRepoSQLite) Update(ctx context.Context, tenant *model.Tenant) (bool, error) {
	res, err := r.DB.ExecContext(ctx, `
UPDATE tenants
SET name = ?, display_name = ?, province = ?, city = ?, district = ?, address = ?, contact_name = ?,
    contact_phone = ?, remark = ?, updated_at = ?
WHERE id = ?`,
		tenant.DisplayName,
		tenant.DisplayName,
		nullableString(tenant.Province),
		nullableString(tenant.City),
		nullableString(tenant.District),
		nullableString(tenant.Address),
		nullableString(tenant.ContactName),
		nullableString(tenant.ContactPhone),
		nullableString(tenant.Remark),
		formatSQLiteTime(tenant.UpdatedAt),
		tenant.ID,
	)
	if err != nil {
		return false, mapTenantConstraintError(err)
	}
	n, _ := res.RowsAffected()
	return n > 0, nil
}

func (r *TenantRepoSQLite) ToggleStatus(ctx context.Context, tenantID, status string) (bool, error) {
	tx, err := r.DB.BeginTx(ctx, nil)
	if err != nil {
		return false, err
	}
	defer tx.Rollback()
	res, err := tx.ExecContext(ctx, `UPDATE tenants SET status = ?, updated_at = ? WHERE id = ?`, status, formatSQLiteTime(nowUTC()), tenantID)
	if err != nil {
		return false, err
	}
	n, _ := res.RowsAffected()
	if n == 0 {
		return false, tx.Commit()
	}
	if _, err := tx.ExecContext(ctx, `UPDATE users SET status = ? WHERE tenant_id = ?`, status, tenantID); err != nil {
		return false, err
	}
	return true, tx.Commit()
}

func (r *TenantRepoSQLite) HasTenantAdmin(ctx context.Context, tenantID string) (bool, error) {
	var exists int
	if err := r.DB.QueryRowContext(ctx, `
SELECT EXISTS(
    SELECT 1
    FROM users u
    JOIN user_credentials c ON c.user_id = u.id
    WHERE u.tenant_id = ? AND u.role = 'tenant_admin'
)`, tenantID).Scan(&exists); err != nil {
		return false, err
	}
	return exists != 0, nil
}

func (r *TenantRepoSQLite) DisplayNameExists(ctx context.Context, displayName string) (bool, error) {
	var exists int
	if err := r.DB.QueryRowContext(ctx, `SELECT EXISTS(SELECT 1 FROM tenants WHERE display_name = ?)`, displayName).Scan(&exists); err != nil {
		return false, err
	}
	return exists != 0, nil
}

func (r *TenantRepoSQLite) TenantAdminAccountExists(ctx context.Context, account string) (bool, error) {
	var exists int
	if err := r.DB.QueryRowContext(ctx, `SELECT EXISTS(SELECT 1 FROM user_credentials WHERE account = ?)`, account).Scan(&exists); err != nil {
		return false, err
	}
	return exists != 0, nil
}

func (r *TenantRepoSQLite) TenantAdminPhoneExists(ctx context.Context, phone string) (bool, error) {
	var exists int
	if err := r.DB.QueryRowContext(ctx, `SELECT EXISTS(SELECT 1 FROM users WHERE phone = ?)`, phone).Scan(&exists); err != nil {
		return false, err
	}
	return exists != 0, nil
}

func (r *TenantRepoSQLite) GetTenantAdminByTenantID(ctx context.Context, tenantID string) (*model.User, string, error) {
	row := r.DB.QueryRowContext(ctx, `
SELECT u.id, COALESCE(u.tenant_id, ''), u.name, COALESCE(u.phone, ''), u.role, u.scope_type,
       u.token_version, u.must_change_password, COALESCE(u.password_updated_at, ''), c.account
FROM users u
JOIN user_credentials c ON c.user_id = u.id
WHERE u.tenant_id = ? AND u.role = 'tenant_admin' AND u.status = 'active'
ORDER BY u.created_at ASC, u.id ASC
LIMIT 1`, tenantID)
	return scanTenantAdmin(row)
}

func (r *TenantRepoSQLite) GetTenantAdminByUserID(ctx context.Context, adminUserID string) (*model.User, string, error) {
	row := r.DB.QueryRowContext(ctx, `
SELECT u.id, COALESCE(u.tenant_id, ''), u.name, COALESCE(u.phone, ''), u.role, u.scope_type,
       u.token_version, u.must_change_password, COALESCE(u.password_updated_at, ''), c.account
FROM users u
JOIN user_credentials c ON c.user_id = u.id
WHERE u.id = ? AND u.role = 'tenant_admin' AND u.status = 'active'
LIMIT 1`, adminUserID)
	return scanTenantAdmin(row)
}

func (r *TenantRepoSQLite) UpdateTenantAdminIdentity(ctx context.Context, adminUserID, account, adminName, phone string) error {
	tx, err := r.DB.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer tx.Rollback()
	if _, err := tx.ExecContext(ctx, `UPDATE users SET name = ?, phone = ? WHERE id = ?`, adminName, nullableString(phone), adminUserID); err != nil {
		return mapTenantConstraintError(err)
	}
	if _, err := tx.ExecContext(ctx, `UPDATE user_credentials SET account = ? WHERE user_id = ?`, account, adminUserID); err != nil {
		return mapTenantConstraintError(err)
	}
	return tx.Commit()
}

func (r *TenantRepoSQLite) ResetTenantAdminPassword(ctx context.Context, adminUserID, passwordHash string) error {
	tx, err := r.DB.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer tx.Rollback()
	if _, err := tx.ExecContext(ctx, `UPDATE user_credentials SET password_hash = ? WHERE user_id = ?`, passwordHash, adminUserID); err != nil {
		return err
	}
	if _, err := tx.ExecContext(ctx, `
UPDATE users
SET must_change_password = 1, password_updated_at = ?, token_version = token_version + 1
WHERE id = ?`, formatSQLiteTime(nowUTC()), adminUserID); err != nil {
		return err
	}
	return tx.Commit()
}

func buildTenantFilter(filter port.TenantFilter) (string, []any) {
	clauses := []string{"1=1"}
	args := []any{}
	if v := strings.TrimSpace(filter.Keyword); v != "" {
		clauses = append(clauses, `LOWER(t.display_name) LIKE '%' || LOWER(?) || '%'`)
		args = append(args, v)
	}
	if v := strings.TrimSpace(filter.Province); v != "" {
		clauses = append(clauses, `COALESCE(t.province, '') = ?`)
		args = append(args, v)
	}
	if v := strings.TrimSpace(filter.City); v != "" {
		clauses = append(clauses, `COALESCE(t.city, '') = ?`)
		args = append(args, v)
	}
	if v := strings.TrimSpace(filter.District); v != "" {
		clauses = append(clauses, `COALESCE(t.district, '') = ?`)
		args = append(args, v)
	}
	if v := strings.TrimSpace(filter.Status); v != "" {
		clauses = append(clauses, `t.status = ?`)
		args = append(args, v)
	}
	return " WHERE " + strings.Join(clauses, " AND "), args
}

type tenantAdminScanner interface {
	Scan(dest ...any) error
}

func scanTenantAdmin(row tenantAdminScanner) (*model.User, string, error) {
	var (
		user              model.User
		account           string
		passwordUpdatedAt sql.NullString
		mustChange        int
	)
	if err := row.Scan(
		&user.ID,
		&user.TenantID,
		&user.Name,
		&user.Phone,
		&user.Role,
		&user.ScopeType,
		&user.TokenVersion,
		&mustChange,
		&passwordUpdatedAt,
		&account,
	); err != nil {
		return nil, "", mapNotFound(err)
	}
	user.MustChangePassword = intToBool(mustChange)
	parsed, err := parseNullableSQLiteTime(passwordUpdatedAt)
	if err != nil {
		return nil, "", err
	}
	user.PasswordUpdatedAt = parsed
	return &user, account, nil
}
