package sqlite

import (
	"context"
	"database/sql"
	"strings"

	"service/internal/domain/model"
	"service/internal/domain/port"
)

type AuditRepoSQLite struct {
	DB *sql.DB
}

func (r *AuditRepoSQLite) Create(ctx context.Context, log *model.AuditLog) error {
	beforeJSON, err := marshalOptionalJSONMap(log.BeforeJSON)
	if err != nil {
		return err
	}
	afterJSON, err := marshalOptionalJSONMap(log.AfterJSON)
	if err != nil {
		return err
	}
	changedFields, err := marshalStringSlice(log.ChangedFields)
	if err != nil {
		return err
	}
	_, err = r.DB.ExecContext(ctx, `
INSERT INTO audit_logs (
	id, request_id, operator_user_id, operator_role, operator_tenant_id, operator_username,
	operator_display_name, target_type, target_id, target_name, action, module, result, error_code,
	before_json, after_json, changed_fields, ip, user_agent, created_at
) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`,
		log.ID,
		log.RequestID,
		nullableString(log.OperatorUserID),
		nullableString(log.OperatorRole),
		nullableString(log.OperatorTenantID),
		log.OperatorUsername,
		log.OperatorDisplayName,
		log.TargetType,
		nullableString(log.TargetID),
		nullableString(log.TargetName),
		log.Action,
		log.Module,
		log.Result,
		nullableString(log.ErrorCode),
		nullableString(beforeJSON),
		nullableString(afterJSON),
		changedFields,
		nullableString(log.IP),
		nullableString(log.UserAgent),
		formatSQLiteTime(log.CreatedAt),
	)
	return err
}

func (r *AuditRepoSQLite) ListPage(ctx context.Context, filter port.AuditFilter) ([]*model.AuditLog, int, error) {
	where, args := buildAuditFilter(filter)
	total := -1
	if filter.NeedTotal {
		if err := r.DB.QueryRowContext(ctx, `SELECT COUNT(1) FROM audit_logs a`+where, args...).Scan(&total); err != nil {
			return nil, 0, err
		}
	}
	page, pageSize := normalizePage(filter.Page, filter.PageSize)
	offset := (page - 1) * pageSize
	args = append(args, pageSize, offset)
	rows, err := r.DB.QueryContext(ctx, `
SELECT id, request_id, COALESCE(operator_user_id, ''), COALESCE(operator_role, ''), COALESCE(operator_tenant_id, ''),
       operator_username, operator_display_name, target_type, COALESCE(target_id, ''), COALESCE(target_name, ''),
       action, module, result, COALESCE(error_code, ''), COALESCE(before_json, ''), COALESCE(after_json, ''),
       changed_fields, COALESCE(ip, ''), COALESCE(user_agent, ''), created_at
FROM audit_logs a`+where+`
ORDER BY created_at DESC, id DESC
LIMIT ? OFFSET ?`, args...)
	if err != nil {
		return nil, 0, err
	}
	defer rows.Close()
	out := []*model.AuditLog{}
	for rows.Next() {
		log, err := scanAuditLog(rows)
		if err != nil {
			return nil, 0, err
		}
		out = append(out, log)
	}
	return out, total, rows.Err()
}

func (r *AuditRepoSQLite) GetByID(ctx context.Context, id string) (*model.AuditLog, error) {
	row := r.DB.QueryRowContext(ctx, `
SELECT id, request_id, COALESCE(operator_user_id, ''), COALESCE(operator_role, ''), COALESCE(operator_tenant_id, ''),
       operator_username, operator_display_name, target_type, COALESCE(target_id, ''), COALESCE(target_name, ''),
       action, module, result, COALESCE(error_code, ''), COALESCE(before_json, ''), COALESCE(after_json, ''),
       changed_fields, COALESCE(ip, ''), COALESCE(user_agent, ''), created_at
FROM audit_logs
WHERE id = ?`, id)
	return scanAuditLog(row)
}

func buildAuditFilter(filter port.AuditFilter) (string, []any) {
	clauses := []string{"1=1"}
	args := []any{}
	if v := strings.TrimSpace(filter.Module); v != "" {
		clauses = append(clauses, `a.module = ?`)
		args = append(args, v)
	}
	if v := strings.TrimSpace(filter.Action); v != "" {
		clauses = append(clauses, `a.action = ?`)
		args = append(args, v)
	}
	if v := strings.TrimSpace(filter.Result); v != "" {
		clauses = append(clauses, `a.result = ?`)
		args = append(args, v)
	}
	if v := strings.TrimSpace(filter.OperatorUserID); v != "" {
		clauses = append(clauses, `COALESCE(a.operator_user_id, '') = ?`)
		args = append(args, v)
	}
	if v := strings.TrimSpace(filter.TargetType); v != "" {
		clauses = append(clauses, `a.target_type = ?`)
		args = append(args, v)
	}
	if v := strings.TrimSpace(filter.TargetID); v != "" {
		clauses = append(clauses, `COALESCE(a.target_id, '') = ?`)
		args = append(args, v)
	}
	if v := strings.TrimSpace(filter.RequestID); v != "" {
		clauses = append(clauses, `a.request_id = ?`)
		args = append(args, v)
	}
	if v := strings.TrimSpace(filter.TenantID); v != "" {
		clauses = append(clauses, `COALESCE(a.operator_tenant_id, '') = ?`)
		args = append(args, v)
	}
	if v := strings.TrimSpace(filter.DateFrom); v != "" {
		clauses = append(clauses, `a.created_at >= ?`)
		args = append(args, v)
	}
	if v := strings.TrimSpace(filter.DateTo); v != "" {
		clauses = append(clauses, `a.created_at <= ?`)
		args = append(args, v)
	}
	if v := strings.TrimSpace(filter.Keyword); v != "" {
		clauses = append(clauses, `(LOWER(COALESCE(a.target_name, '')) LIKE '%' || LOWER(?) || '%' OR LOWER(COALESCE(a.target_id, '')) LIKE '%' || LOWER(?) || '%')`)
		args = append(args, v, v)
	}
	return " WHERE " + strings.Join(clauses, " AND "), args
}

type auditLogScanner interface {
	Scan(dest ...any) error
}

func scanAuditLog(row auditLogScanner) (*model.AuditLog, error) {
	var (
		log                            model.AuditLog
		beforeJSON, afterJSON, changed sql.NullString
		createdAt                      string
	)
	if err := row.Scan(
		&log.ID,
		&log.RequestID,
		&log.OperatorUserID,
		&log.OperatorRole,
		&log.OperatorTenantID,
		&log.OperatorUsername,
		&log.OperatorDisplayName,
		&log.TargetType,
		&log.TargetID,
		&log.TargetName,
		&log.Action,
		&log.Module,
		&log.Result,
		&log.ErrorCode,
		&beforeJSON,
		&afterJSON,
		&changed,
		&log.IP,
		&log.UserAgent,
		&createdAt,
	); err != nil {
		return nil, mapNotFound(err)
	}
	var err error
	log.BeforeJSON, err = unmarshalOptionalJSONMap(beforeJSON)
	if err != nil {
		return nil, err
	}
	log.AfterJSON, err = unmarshalOptionalJSONMap(afterJSON)
	if err != nil {
		return nil, err
	}
	log.ChangedFields, err = unmarshalStringSlice(changed)
	if err != nil {
		return nil, err
	}
	log.CreatedAt, err = parseSQLiteTime(createdAt)
	if err != nil {
		return nil, err
	}
	return &log, nil
}
