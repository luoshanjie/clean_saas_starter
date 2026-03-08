package usecase_test

import (
	"context"
	"testing"

	"service/internal/app/usecase"
	"service/internal/domain/model"
)

type mockPermMe struct{ perms []string }

func (m mockPermMe) Enforce(ctx context.Context, role, permission string) (bool, error) {
	return true, nil
}
func (m mockPermMe) ListByRole(ctx context.Context, role string) ([]string, error) {
	return m.perms, nil
}

func TestAuthMe_ReturnsIdentity(t *testing.T) {
	u := &usecase.AuthMeUsecase{Perm: mockPermMe{perms: []string{"p1"}}}
	in := usecase.AuthMeInput{User: &model.User{ID: "u1", Name: "n", Role: "tenant_member", ScopeType: "tenant", TenantID: "t1", TenantName: "Tenant A"}}
	out := u.Execute(context.Background(), in)
	if out.User.ID != "u1" || out.Role != "tenant_member" || out.ScopeType != "tenant" || out.TenantID != "t1" {
		t.Fatalf("unexpected output: %+v", out)
	}
	if out.TenantName != "Tenant A" {
		t.Fatalf("unexpected tenant_name: %s", out.TenantName)
	}
	if out.Permissions == nil || len(out.Permissions) != 1 {
		t.Fatalf("expected permissions to be non-nil")
	}
}
