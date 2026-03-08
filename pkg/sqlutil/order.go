package sqlutil

import (
	"errors"
	"strings"
)

var (
	ErrInvalidOrderBy = errors.New("invalid order_by")
	ErrInvalidOrder   = errors.New("invalid order")
)

// OrderBy builds a safe ORDER BY clause using a whitelist mapping.
// If field is empty, it returns an empty string (caller may apply a default order).
func OrderBy(allowed map[string]string, field, dir string) (string, error) {
	if field == "" {
		return "", nil
	}
	col, ok := allowed[field]
	if !ok {
		return "", ErrInvalidOrderBy
	}

	order := strings.ToUpper(strings.TrimSpace(dir))
	if order == "" {
		order = "ASC"
	}
	if order != "ASC" && order != "DESC" {
		return "", ErrInvalidOrder
	}

	return " ORDER BY " + col + " " + order, nil
}
