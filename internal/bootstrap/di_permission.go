package bootstrap

import (
	"github.com/casbin/casbin/v2"
	fileadapter "github.com/casbin/casbin/v2/persist/file-adapter"

	casbinrepo "service/internal/repo/casbin"
)

func newPermissionChecker() (*casbinrepo.PermissionChecker, error) {
	enforcer, err := casbin.NewEnforcer(
		"internal/repo/casbin/model.conf",
		fileadapter.NewAdapter("internal/repo/casbin/policy.csv"),
	)
	if err != nil {
		return nil, err
	}
	return &casbinrepo.PermissionChecker{Enforcer: enforcer}, nil
}
