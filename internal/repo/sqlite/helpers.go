package sqlite

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"time"

	"service/internal/domain/authctx"
	domainErr "service/internal/domain/errors"
)

const (
	sqliteTimeLayout = time.RFC3339Nano
)

func formatSQLiteTime(t time.Time) string {
	return t.UTC().Format(sqliteTimeLayout)
}

func nullableSQLiteTime(t *time.Time) any {
	if t == nil {
		return nil
	}
	return formatSQLiteTime(*t)
}

func parseSQLiteTime(v string) (time.Time, error) {
	v = strings.TrimSpace(v)
	if v == "" {
		return time.Time{}, errors.New("empty sqlite time")
	}
	layouts := []string{
		sqliteTimeLayout,
		time.RFC3339,
		"2006-01-02 15:04:05",
		"2006-01-02 15:04:05Z07:00",
	}
	for _, layout := range layouts {
		if t, err := time.Parse(layout, v); err == nil {
			return t, nil
		}
	}
	return time.Time{}, fmt.Errorf("parse sqlite time %q", v)
}

func parseNullableSQLiteTime(v sql.NullString) (*time.Time, error) {
	if !v.Valid || strings.TrimSpace(v.String) == "" {
		return nil, nil
	}
	t, err := parseSQLiteTime(v.String)
	if err != nil {
		return nil, err
	}
	return &t, nil
}

func nullableString(v string) any {
	if strings.TrimSpace(v) == "" {
		return nil
	}
	return strings.TrimSpace(v)
}

func boolToInt(v bool) int {
	if v {
		return 1
	}
	return 0
}

func intToBool(v int) bool {
	return v != 0
}

func mapNotFound(err error) error {
	if errors.Is(err, sql.ErrNoRows) {
		return domainErr.ErrNotFound
	}
	return err
}

func scopeInfo(ctx context.Context) authctx.Info {
	info, _ := authctx.From(ctx)
	return info
}

func isPlatformScope(ctx context.Context) bool {
	return scopeInfo(ctx).ScopeType == "platform"
}

func tenantScopeID(ctx context.Context) string {
	info := scopeInfo(ctx)
	if info.ScopeType == "tenant" {
		return strings.TrimSpace(info.TenantID)
	}
	return ""
}

func marshalOptionalJSONMap(v map[string]any) (string, error) {
	if v == nil {
		return "", nil
	}
	data, err := json.Marshal(v)
	if err != nil {
		return "", err
	}
	if string(data) == "null" {
		return "", nil
	}
	return string(data), nil
}

func unmarshalOptionalJSONMap(v sql.NullString) (map[string]any, error) {
	if !v.Valid || strings.TrimSpace(v.String) == "" || v.String == "null" {
		return nil, nil
	}
	out := map[string]any{}
	if err := json.Unmarshal([]byte(v.String), &out); err != nil {
		return nil, err
	}
	return out, nil
}

func marshalStringSlice(v []string) (string, error) {
	if len(v) == 0 {
		return "[]", nil
	}
	data, err := json.Marshal(v)
	if err != nil {
		return "", err
	}
	return string(data), nil
}

func unmarshalStringSlice(v sql.NullString) ([]string, error) {
	if !v.Valid || strings.TrimSpace(v.String) == "" || v.String == "null" {
		return []string{}, nil
	}
	var out []string
	if err := json.Unmarshal([]byte(v.String), &out); err != nil {
		return nil, err
	}
	if out == nil {
		out = []string{}
	}
	return out, nil
}

func mapTenantConstraintError(err error) error {
	if err == nil {
		return nil
	}
	msg := err.Error()
	switch {
	case strings.Contains(msg, "UNIQUE constraint failed: tenants.display_name"):
		return domainErr.ErrTenantDisplayNameExists
	case strings.Contains(msg, "UNIQUE constraint failed: user_credentials.account"):
		return domainErr.ErrTenantAdminAccountExists
	case strings.Contains(msg, "UNIQUE constraint failed: users.phone"):
		return domainErr.ErrTenantAdminPhoneExists
	default:
		return err
	}
}

func normalizePage(page, pageSize int) (int, int) {
	if page <= 0 {
		page = 1
	}
	switch {
	case pageSize <= 0:
		pageSize = 20
	case pageSize > 100:
		pageSize = 100
	}
	return page, pageSize
}
