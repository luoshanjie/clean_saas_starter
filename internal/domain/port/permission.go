package port

import "context"

type PermissionChecker interface {
	Enforce(ctx context.Context, role, permission string) (bool, error)
	ListByRole(ctx context.Context, role string) ([]string, error)
}
