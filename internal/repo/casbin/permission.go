package casbin

import (
	"context"

	"github.com/casbin/casbin/v2"
)

type PermissionChecker struct {
	Enforcer *casbin.Enforcer
}

func (p *PermissionChecker) Enforce(ctx context.Context, role, permission string) (bool, error) {
	return p.Enforcer.Enforce(role, permission, "allow")
}

func (p *PermissionChecker) ListByRole(ctx context.Context, role string) ([]string, error) {
	policies := p.Enforcer.GetFilteredPolicy(0, role)
	out := make([]string, 0, len(policies))
	for _, p := range policies {
		if len(p) >= 2 {
			out = append(out, p[1])
		}
	}
	return out, nil
}
