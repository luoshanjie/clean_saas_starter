package pg

import (
	"errors"

	"github.com/jackc/pgx/v5"

	domainErr "service/internal/domain/errors"
)

func mapRowNotFound(err error) error {
	if errors.Is(err, pgx.ErrNoRows) {
		return domainErr.ErrNotFound
	}
	return err
}
