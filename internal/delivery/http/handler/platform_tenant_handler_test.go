package handler_test

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/labstack/echo/v4"

	"service/internal/app/usecase"
	"service/internal/delivery/http/handler"
	"service/internal/delivery/http/middleware"
	"service/internal/domain/model"
	"service/internal/domain/port"
)

type tenantRepoForHandler struct {
	created *model.TenantCreateInput
}

func (m *tenantRepoForHandler) CreateWithAdmin(ctx context.Context, in *model.TenantCreateInput) (*model.TenantCreateOutput, error) {
	m.created = in
	return &model.TenantCreateOutput{
		TenantID:           in.TenantID,
		TenantAdminUserID:  in.TenantAdminUserID,
		TenantAdminAccount: in.TenantAdminAccount,
		Status:             in.Status,
		CreatedAt:          in.CreatedAt,
	}, nil
}
func (m *tenantRepoForHandler) ListPage(ctx context.Context, filter port.TenantFilter) ([]*model.TenantListItem, int, error) {
	return []*model.TenantListItem{{
		Tenant:             &model.Tenant{ID: "t1", DisplayName: "Tenant A", Status: "active", CreatedAt: time.Unix(1, 0), UpdatedAt: time.Unix(1, 0)},
		TenantAdminUserID:  "u1",
		TenantAdminAccount: "km_admin",
		TenantAdminName:    "",
		TenantAdminPhone:   "13800138000",
	}}, 1, nil
}
func (m *tenantRepoForHandler) GetByID(ctx context.Context, tenantID string) (*model.Tenant, error) {
	return &model.Tenant{ID: tenantID}, nil
}
func (m *tenantRepoForHandler) Update(ctx context.Context, tenant *model.Tenant) (bool, error) {
	return true, nil
}
func (m *tenantRepoForHandler) ToggleStatus(ctx context.Context, tenantID, status string) (bool, error) {
	return true, nil
}
func (m *tenantRepoForHandler) HasTenantAdmin(ctx context.Context, tenantID string) (bool, error) {
	return true, nil
}
func (m *tenantRepoForHandler) DisplayNameExists(ctx context.Context, displayName string) (bool, error) {
	return false, nil
}
func (m *tenantRepoForHandler) TenantAdminAccountExists(ctx context.Context, adminAccount string) (bool, error) {
	return false, nil
}
func (m *tenantRepoForHandler) TenantAdminPhoneExists(ctx context.Context, adminPhone string) (bool, error) {
	return false, nil
}
func (m *tenantRepoForHandler) GetTenantAdminByTenantID(ctx context.Context, tenantID string) (*model.User, string, error) {
	return &model.User{ID: "u1", Phone: "13800138000"}, "km_admin", nil
}
func (m *tenantRepoForHandler) GetTenantAdminByUserID(ctx context.Context, adminUserID string) (*model.User, string, error) {
	return &model.User{ID: adminUserID, Phone: "13800138000"}, "km_admin", nil
}
func (m *tenantRepoForHandler) UpdateTenantAdminIdentity(ctx context.Context, adminUserID, adminUsername, adminName, adminPhone string) error {
	return nil
}
func (m *tenantRepoForHandler) ResetTenantAdminPassword(ctx context.Context, adminUserID, passwordHash string) error {
	return nil
}

type tenantResp struct {
	Code int `json:"code"`
}

func TestPlatformTenantHandler_Create(t *testing.T) {
	e := echo.New()
	repo := &tenantRepoForHandler{}
	h := &handler.PlatformTenantHandler{
		CreateUC: &usecase.CreatePlatformTenantUsecase{
			Repo:  repo,
			IDGen: func() string { return "id-1" },
			Now:   func() time.Time { return time.Unix(1, 0) },
		},
	}
	body := []byte(`{"display_name":"Tenant A","admin_account":"km_admin","admin_name":"Tenant Admin","admin_phone":"13800138000"}`)
	req := httptest.NewRequest(http.MethodPost, "/platform/tenant/create", bytes.NewReader(body))
	req.Header.Set(echo.HeaderContentType, echo.MIMEApplicationJSON)
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)
	c.Set(middleware.CtxRole, "platform_op")
	c.Set(middleware.CtxRequestID, "rid-1")

	if err := h.Create(c); err != nil {
		t.Fatalf("handler error: %v", err)
	}
	var out tenantResp
	_ = json.Unmarshal(rec.Body.Bytes(), &out)
	if out.Code != 0 {
		t.Fatalf("expected code 0, got %d", out.Code)
	}
	if repo.created == nil || repo.created.DisplayName != "Tenant A" {
		t.Fatalf("unexpected create input: %+v", repo.created)
	}
}

func TestPlatformTenantHandler_List(t *testing.T) {
	e := echo.New()
	h := &handler.PlatformTenantHandler{
		ListUC: &usecase.ListPlatformTenantsUsecase{Repo: &tenantRepoForHandler{}},
	}
	req := httptest.NewRequest(http.MethodPost, "/platform/tenant/list", bytes.NewReader([]byte(`{"page":1,"page_size":20}`)))
	req.Header.Set(echo.HeaderContentType, echo.MIMEApplicationJSON)
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)
	c.Set(middleware.CtxRole, "platform_op")
	c.Set(middleware.CtxRequestID, "rid-2")

	if err := h.List(c); err != nil {
		t.Fatalf("handler error: %v", err)
	}
	var out tenantResp
	_ = json.Unmarshal(rec.Body.Bytes(), &out)
	if out.Code != 0 {
		t.Fatalf("expected code 0, got %d", out.Code)
	}
	var payload struct {
		Data struct {
			Items []map[string]any `json:"items"`
		} `json:"data"`
	}
	_ = json.Unmarshal(rec.Body.Bytes(), &payload)
	if len(payload.Data.Items) != 1 {
		t.Fatalf("expected one list item")
	}
	name, _ := payload.Data.Items[0]["tenant_admin_name"].(string)
	if name != "" {
		t.Fatalf("expected empty tenant_admin_name, got %q", name)
	}
}

func TestPlatformTenantHandler_CheckDisplayName(t *testing.T) {
	e := echo.New()
	h := &handler.PlatformTenantHandler{
		CheckDisplayNameUC: &usecase.CheckPlatformTenantDisplayNameUsecase{Repo: &tenantRepoForHandler{}},
	}
	req := httptest.NewRequest(http.MethodPost, "/platform/tenant/check-display-name", bytes.NewReader([]byte(`{"display_name":"Tenant A"}`)))
	req.Header.Set(echo.HeaderContentType, echo.MIMEApplicationJSON)
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)
	c.Set(middleware.CtxRole, "platform_op")
	c.Set(middleware.CtxRequestID, "rid-3")

	if err := h.CheckDisplayName(c); err != nil {
		t.Fatalf("handler error: %v", err)
	}
	var out tenantResp
	_ = json.Unmarshal(rec.Body.Bytes(), &out)
	if out.Code != 0 {
		t.Fatalf("expected code 0, got %d", out.Code)
	}
}

func TestPlatformTenantHandler_CheckAdminAccount(t *testing.T) {
	e := echo.New()
	h := &handler.PlatformTenantHandler{
		CheckAdminAccountUC: &usecase.CheckPlatformTenantAdminAccountUsecase{Repo: &tenantRepoForHandler{}},
	}
	req := httptest.NewRequest(http.MethodPost, "/platform/tenant/check-admin-account", bytes.NewReader([]byte(`{"admin_account":"km_admin"}`)))
	req.Header.Set(echo.HeaderContentType, echo.MIMEApplicationJSON)
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)
	c.Set(middleware.CtxRole, "platform_op")
	c.Set(middleware.CtxRequestID, "rid-4")

	if err := h.CheckAdminAccount(c); err != nil {
		t.Fatalf("handler error: %v", err)
	}
	var out tenantResp
	_ = json.Unmarshal(rec.Body.Bytes(), &out)
	if out.Code != 0 {
		t.Fatalf("expected code 0, got %d", out.Code)
	}
}

func TestPlatformTenantHandler_CheckAdminPhone(t *testing.T) {
	e := echo.New()
	h := &handler.PlatformTenantHandler{
		CheckAdminPhoneUC: &usecase.CheckPlatformTenantAdminPhoneUsecase{Repo: &tenantRepoForHandler{}},
	}
	req := httptest.NewRequest(http.MethodPost, "/platform/tenant/check-admin-phone", bytes.NewReader([]byte(`{"admin_phone":"13800138000"}`)))
	req.Header.Set(echo.HeaderContentType, echo.MIMEApplicationJSON)
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)
	c.Set(middleware.CtxRole, "platform_op")
	c.Set(middleware.CtxRequestID, "rid-5")

	if err := h.CheckAdminPhone(c); err != nil {
		t.Fatalf("handler error: %v", err)
	}
	var out tenantResp
	_ = json.Unmarshal(rec.Body.Bytes(), &out)
	if out.Code != 0 {
		t.Fatalf("expected code 0, got %d", out.Code)
	}
}

func TestPlatformTenantHandler_ChangeAdmin(t *testing.T) {
	e := echo.New()
	h := &handler.PlatformTenantHandler{
		ChangeAdminUC: &usecase.ChangePlatformTenantAdminUsecase{Repo: &tenantRepoForHandler{}},
	}
	req := httptest.NewRequest(http.MethodPost, "/platform/tenant/change-admin", bytes.NewReader([]byte(`{"tenant_id":"t1","admin_account":"km_admin2","admin_name":"Tenant Admin B","admin_phone":"13900139000"}`)))
	req.Header.Set(echo.HeaderContentType, echo.MIMEApplicationJSON)
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)
	c.Set(middleware.CtxRole, "platform_op")
	c.Set(middleware.CtxRequestID, "rid-6")

	if err := h.ChangeAdmin(c); err != nil {
		t.Fatalf("handler error: %v", err)
	}
	var out tenantResp
	_ = json.Unmarshal(rec.Body.Bytes(), &out)
	if out.Code != 0 {
		t.Fatalf("expected code 0, got %d", out.Code)
	}
}

func TestPlatformTenantHandler_ResetAdminPassword(t *testing.T) {
	e := echo.New()
	h := &handler.PlatformTenantHandler{
		ResetPasswordUC: &usecase.ResetPlatformTenantAdminPasswordUsecase{
			Repo: &tenantRepoForHandler{},
			PassGen: func() string {
				return "Temp12345"
			},
		},
	}
	req := httptest.NewRequest(http.MethodPost, "/platform/tenant/admin/reset-password", bytes.NewReader([]byte(`{"tenant_id":"t1"}`)))
	req.Header.Set(echo.HeaderContentType, echo.MIMEApplicationJSON)
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)
	c.Set(middleware.CtxRole, "platform_op")
	c.Set(middleware.CtxRequestID, "rid-7")

	if err := h.ResetAdminPassword(c); err != nil {
		t.Fatalf("handler error: %v", err)
	}
	var out tenantResp
	_ = json.Unmarshal(rec.Body.Bytes(), &out)
	if out.Code != 0 {
		t.Fatalf("expected code 0, got %d", out.Code)
	}
}
