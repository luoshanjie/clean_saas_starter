package usecase

import (
	"strings"

	"github.com/google/uuid"
)

func canonicalUUIDText(id string) (string, error) {
	s := strings.TrimSpace(id)
	if parsed, err := uuid.Parse(s); err == nil {
		return parsed.String(), nil
	}
	compact := strings.ReplaceAll(s, "-", "")
	if len(compact) == 32 {
		parsed, err := uuid.Parse(compact)
		if err != nil {
			return "", err
		}
		return parsed.String(), nil
	}
	return s, nil
}
