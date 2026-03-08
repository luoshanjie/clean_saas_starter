//go:build integration

package integration

import (
	"bytes"
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"
	"time"

	"golang.org/x/crypto/bcrypt"

	"service/internal/bootstrap"
	"service/pkg/logger"
)

type tenantCreateResp struct {
	Code    int    `json:"code"`
	Message string `json:"message"`
	Data    struct {
		TenantID           string `json:"tenant_id"`
		TenantAdminUserID  string `json:"tenant_admin_user_id"`
		TenantAdminAccount string `json:"tenant_admin_account"`
		TenantAdminName    string `json:"tenant_admin_name"`
		Status             string `json:"status"`
	} `json:"data"`
}

type tenantListResp struct {
	Code int `json:"code"`
	Data struct {
		Items []struct {
			TenantID           string `json:"tenant_id"`
			DisplayName        string `json:"display_name"`
			TenantAdminAccount string `json:"tenant_admin_account"`
			Status             string `json:"status"`
		} `json:"items"`
		Total int `json:"total"`
	} `json:"data"`
}

type statusResp struct {
	Code int `json:"code"`
	Data struct {
		Status string `json:"status"`
	} `json:"data"`
}

type updateDisplayNameResp struct {
	Code int `json:"code"`
	Data struct {
		Status string `json:"status"`
		Name   string `json:"name"`
	} `json:"data"`
}

type changePhoneChallengeResp struct {
	Code int `json:"code"`
	Data struct {
		ChallengeID    string `json:"challenge_id"`
		MaskedNewPhone string `json:"masked_new_phone"`
		ExpiresIn      int    `json:"expires_in"`
		ResendIn       int    `json:"resend_in"`
	} `json:"data"`
}

type changePhoneVerifyResp struct {
	Code int `json:"code"`
	Data struct {
		Status       string `json:"status"`
		ForceRelogin bool   `json:"force_relogin"`
	} `json:"data"`
}

type resetAdminPasswordResp struct {
	Code int `json:"code"`
	Data struct {
		AdminUserID        string `json:"admin_user_id"`
		TemporaryPassword  string `json:"temporary_password"`
		MustChangePassword bool   `json:"must_change_password"`
	} `json:"data"`
}

func TestSQLiteFlow_Integration(t *testing.T) {
	chdirToRepoRoot(t)

	ctx := context.Background()
	dbPath := filepath.Join(t.TempDir(), "integration.db")
	runtime, err := bootstrap.InitDB(ctx, bootstrap.Config{
		DBDriver:   bootstrap.DBDriverSQLite,
		SQLitePath: dbPath,
	})
	if err != nil {
		t.Fatalf("InitDB sqlite: %v", err)
	}
	defer runtime.Close()

	applySQLiteSchema(t, runtime.SQLite)
	seedSQLitePlatformUser(t, runtime.SQLite)

	e := bootstrap.NewEcho(logger.NewNopLogger())
	_, err = bootstrap.Build(ctx, e, runtime, fixedIDGen(
		"10000000-0000-0000-0000-000000000001",
		"10000000-0000-0000-0000-000000000002",
		"10000000-0000-0000-0000-000000000003",
	), time.Now, bootstrap.Config{
		DBDriver:   bootstrap.DBDriverSQLite,
		SQLitePath: dbPath,
		JWTSecret:  "test_secret",
	}, logger.NewNopLogger())
	if err != nil {
		t.Fatalf("build sqlite app: %v", err)
	}
	login := doJSONRequest(t, e, http.MethodPost, "/api/v1/auth/login", "", map[string]any{
		"account":  "platform_root",
		"password": "pass123",
	})
	var lr loginChallengeResp
	decodeRecorder(t, login, &lr)
	if lr.Code != 0 || lr.Data.ChallengeID == "" {
		t.Fatalf("login failed: %+v", lr)
	}

	verify := doJSONRequest(t, e, http.MethodPost, "/api/v1/auth/login/verify", "", map[string]any{
		"challenge_id": lr.Data.ChallengeID,
		"otp_code":     "123456",
	})
	var lv loginVerifyResp
	decodeRecorder(t, verify, &lv)
	if lv.Code != 0 || lv.Data.AccessToken == "" {
		t.Fatalf("verify failed: %+v", lv)
	}

	me := doJSONRequest(t, e, http.MethodPost, "/api/v1/auth/me", lv.Data.AccessToken, map[string]any{})
	var mr meResp
	decodeRecorder(t, me, &mr)
	if mr.Code != 0 || mr.Data.User.ID == "" {
		t.Fatalf("me failed: %+v", mr)
	}

	refresh := doJSONRequest(t, e, http.MethodPost, "/api/v1/auth/refresh", "", map[string]any{
		"refresh_token": lv.Data.RefreshToken,
	})
	var rr refreshResp
	decodeRecorder(t, refresh, &rr)
	if rr.Code != 0 || rr.Data.AccessToken == "" || rr.Data.RefreshToken == "" {
		t.Fatalf("refresh failed: %+v", rr)
	}

	create := doJSONRequest(t, e, http.MethodPost, "/api/v1/platform/tenant/create", lv.Data.AccessToken, map[string]any{
		"display_name":  "Acme Cloud",
		"province":      "Shanghai",
		"city":          "Shanghai",
		"district":      "Pudong",
		"address":       "Century Ave 1",
		"contact_name":  "Alice",
		"contact_phone": "13800000011",
		"status":        "active",
		"admin_account": "acme_admin",
		"admin_name":    "Acme Admin",
		"admin_phone":   "13800000012",
		"remark":        "sqlite integration",
	})
	var cr tenantCreateResp
	decodeRecorder(t, create, &cr)
	if cr.Code != 0 || cr.Data.TenantID == "" || cr.Data.TenantAdminUserID == "" {
		t.Fatalf("tenant create failed: %+v", cr)
	}

	list := doJSONRequest(t, e, http.MethodPost, "/api/v1/platform/tenant/list", lv.Data.AccessToken, map[string]any{
		"keyword":    "Acme",
		"need_total": true,
		"page":       1,
		"page_size":  10,
	})
	var tr tenantListResp
	decodeRecorder(t, list, &tr)
	if tr.Code != 0 {
		t.Fatalf("tenant list failed: %+v", tr)
	}
	if tr.Data.Total != 1 || len(tr.Data.Items) != 1 {
		t.Fatalf("unexpected tenant list payload: %+v", tr)
	}
	if tr.Data.Items[0].DisplayName != "Acme Cloud" || tr.Data.Items[0].TenantAdminAccount != "acme_admin" {
		t.Fatalf("unexpected tenant list item: %+v", tr.Data.Items[0])
	}

	updateDisplayName := doJSONRequest(t, e, http.MethodPost, "/api/v1/auth/profile/update-name", lv.Data.AccessToken, map[string]any{
		"name": "Platform Owner",
	})
	var udn updateDisplayNameResp
	decodeRecorder(t, updateDisplayName, &udn)
	if udn.Code != 0 || udn.Data.Status != "ok" || udn.Data.Name != "Platform Owner" {
		t.Fatalf("update display name failed: %+v", udn)
	}

	meAfterDisplayNameUpdate := doJSONRequest(t, e, http.MethodPost, "/api/v1/auth/me", lv.Data.AccessToken, map[string]any{})
	var mrAfterDisplayNameUpdate meResp
	decodeRecorder(t, meAfterDisplayNameUpdate, &mrAfterDisplayNameUpdate)
	if mrAfterDisplayNameUpdate.Code != 0 || mrAfterDisplayNameUpdate.Data.User.ID == "" {
		t.Fatalf("me after display name update failed: %+v", mrAfterDisplayNameUpdate)
	}
	if mrAfterDisplayNameUpdate.Data.User.Name != "Platform Owner" {
		t.Fatalf("unexpected display name after update: %+v", mrAfterDisplayNameUpdate)
	}

	resetAdminPassword := doJSONRequest(t, e, http.MethodPost, "/api/v1/platform/tenant/admin/reset-password", lv.Data.AccessToken, map[string]any{
		"tenant_id": cr.Data.TenantID,
	})
	var rap resetAdminPasswordResp
	decodeRecorder(t, resetAdminPassword, &rap)
	if rap.Code != 0 || rap.Data.AdminUserID == "" || rap.Data.TemporaryPassword == "" || !rap.Data.MustChangePassword {
		t.Fatalf("reset tenant admin password failed: %+v", rap)
	}

	tenantAdminLogin := doJSONRequest(t, e, http.MethodPost, "/api/v1/auth/login", "", map[string]any{
		"account":  "acme_admin",
		"password": rap.Data.TemporaryPassword,
	})
	var tal loginChallengeResp
	decodeRecorder(t, tenantAdminLogin, &tal)
	if tal.Code != 0 || tal.Data.ChallengeID == "" {
		t.Fatalf("tenant admin login failed: %+v", tal)
	}

	tenantAdminVerify := doJSONRequest(t, e, http.MethodPost, "/api/v1/auth/login/verify", "", map[string]any{
		"challenge_id": tal.Data.ChallengeID,
		"otp_code":     "123456",
	})
	var tav loginVerifyResp
	decodeRecorder(t, tenantAdminVerify, &tav)
	if tav.Code != 0 || tav.Data.AccessToken == "" || !tav.Data.MustChangePassword {
		t.Fatalf("tenant admin verify failed: %+v", tav)
	}

	tenantAdminMe := doJSONRequest(t, e, http.MethodPost, "/api/v1/auth/me", tav.Data.AccessToken, map[string]any{})
	var tam meResp
	decodeRecorder(t, tenantAdminMe, &tam)
	if tam.Code != 0 || tam.Data.User.ID == "" {
		t.Fatalf("tenant admin me failed: %+v", tam)
	}
	if !tam.Data.MustChangePassword {
		t.Fatalf("tenant admin should require password change on first login: %+v", tam)
	}

	changePassword := doJSONRequest(t, e, http.MethodPost, "/api/v1/auth/change-password", lv.Data.AccessToken, map[string]any{
		"old_password": "pass123",
		"new_password": "pass12345",
	})
	var cp statusResp
	decodeRecorder(t, changePassword, &cp)
	if cp.Code != 0 || cp.Data.Status != "ok" {
		t.Fatalf("change password failed: %+v", cp)
	}

	meAfterPasswordChange := doJSONRequest(t, e, http.MethodPost, "/api/v1/auth/me", lv.Data.AccessToken, map[string]any{})
	var mrAfterPasswordChange meResp
	decodeRecorder(t, meAfterPasswordChange, &mrAfterPasswordChange)
	if mrAfterPasswordChange.Code == 0 {
		t.Fatalf("old access token should be invalid after password change")
	}

	login2 := doJSONRequest(t, e, http.MethodPost, "/api/v1/auth/login", "", map[string]any{
		"account":  "platform_root",
		"password": "pass12345",
	})
	var lr2 loginChallengeResp
	decodeRecorder(t, login2, &lr2)
	if lr2.Code != 0 || lr2.Data.ChallengeID == "" {
		t.Fatalf("login with new password failed: %+v", lr2)
	}

	verify2 := doJSONRequest(t, e, http.MethodPost, "/api/v1/auth/login/verify", "", map[string]any{
		"challenge_id": lr2.Data.ChallengeID,
		"otp_code":     "123456",
	})
	var lv2 loginVerifyResp
	decodeRecorder(t, verify2, &lv2)
	if lv2.Code != 0 || lv2.Data.AccessToken == "" {
		t.Fatalf("verify after password change failed: %+v", lv2)
	}

	changePhoneChallenge := doJSONRequest(t, e, http.MethodPost, "/api/v1/auth/change-phone/challenge", lv2.Data.AccessToken, map[string]any{
		"new_phone":    "13800000099",
		"old_password": "pass12345",
	})
	var cpc changePhoneChallengeResp
	decodeRecorder(t, changePhoneChallenge, &cpc)
	if cpc.Code != 0 || cpc.Data.ChallengeID == "" {
		t.Fatalf("change phone challenge failed: %+v", cpc)
	}

	forceSQLitePhoneChallengeResendable(t, runtime.SQLite, cpc.Data.ChallengeID)

	changePhoneResend := doJSONRequest(t, e, http.MethodPost, "/api/v1/auth/change-phone/resend", lv2.Data.AccessToken, map[string]any{
		"challenge_id": cpc.Data.ChallengeID,
	})
	var cpr changePhoneChallengeResp
	decodeRecorder(t, changePhoneResend, &cpr)
	if cpr.Code != 0 || cpr.Data.ChallengeID != cpc.Data.ChallengeID {
		t.Fatalf("change phone resend failed: %+v", cpr)
	}

	changePhoneVerify := doJSONRequest(t, e, http.MethodPost, "/api/v1/auth/change-phone/verify", lv2.Data.AccessToken, map[string]any{
		"challenge_id": cpc.Data.ChallengeID,
		"otp_code":     "123456",
	})
	var cpv changePhoneVerifyResp
	decodeRecorder(t, changePhoneVerify, &cpv)
	if cpv.Code != 0 || cpv.Data.Status != "ok" || !cpv.Data.ForceRelogin {
		t.Fatalf("change phone verify failed: %+v", cpv)
	}

	meAfterPhoneChange := doJSONRequest(t, e, http.MethodPost, "/api/v1/auth/me", lv2.Data.AccessToken, map[string]any{})
	var mrAfterPhoneChange meResp
	decodeRecorder(t, meAfterPhoneChange, &mrAfterPhoneChange)
	if mrAfterPhoneChange.Code == 0 {
		t.Fatalf("old access token should be invalid after phone change")
	}

	login3 := doJSONRequest(t, e, http.MethodPost, "/api/v1/auth/login", "", map[string]any{
		"account":  "platform_root",
		"password": "pass12345",
	})
	var lr3 loginChallengeResp
	decodeRecorder(t, login3, &lr3)
	if lr3.Code != 0 || lr3.Data.ChallengeID == "" {
		t.Fatalf("login after phone change failed: %+v", lr3)
	}
	if lr3.Data.MaskedPhone != "138****0099" {
		t.Fatalf("unexpected masked phone after phone change: %+v", lr3)
	}
}

func applySQLiteSchema(t *testing.T, db *sql.DB) {
	t.Helper()
	schema, err := os.ReadFile(filepath.Join("migrations", "sqlite", "demo_schema_init.sql"))
	if err != nil {
		t.Fatalf("read sqlite schema: %v", err)
	}
	if _, err := db.Exec(string(schema)); err != nil {
		t.Fatalf("apply sqlite schema: %v", err)
	}
}

func seedSQLitePlatformUser(t *testing.T, db *sql.DB) {
	t.Helper()
	hash, err := bcrypt.GenerateFromPassword([]byte("pass123"), bcrypt.DefaultCost)
	if err != nil {
		t.Fatalf("bcrypt: %v", err)
	}
	now := time.Now().UTC().Format(time.RFC3339Nano)
	_, err = db.Exec(`
INSERT INTO users (
	id, tenant_id, name, phone, role, scope_type, status, token_version, must_change_password,
	password_updated_at, created_at, updated_at
) VALUES (?, NULL, ?, ?, 'platform_op', 'platform', 'active', 0, 0, NULL, ?, ?)`,
		"00000000-0000-0000-0000-000000000001",
		"Platform Root",
		"13800000001",
		now,
		now,
	)
	if err != nil {
		t.Fatalf("insert sqlite platform user: %v", err)
	}
	_, err = db.Exec(`INSERT INTO user_credentials (user_id, account, password_hash, created_at) VALUES (?, ?, ?, ?)`,
		"00000000-0000-0000-0000-000000000001",
		"platform_root",
		string(hash),
		now,
	)
	if err != nil {
		t.Fatalf("insert sqlite platform credential: %v", err)
	}
}

func forceSQLitePhoneChallengeResendable(t *testing.T, db *sql.DB, challengeID string) {
	t.Helper()
	lastSentAt := time.Now().UTC().Add(-2 * time.Minute).Format(time.RFC3339Nano)
	if _, err := db.Exec(`UPDATE phone_change_challenges SET last_sent_at = ? WHERE id = ?`, lastSentAt, challengeID); err != nil {
		t.Fatalf("update sqlite phone change challenge resend window: %v", err)
	}
}

func fixedIDGen(ids ...string) func() string {
	index := 0
	return func() string {
		if index < len(ids) {
			id := ids[index]
			index++
			return id
		}
		id := "10000000-0000-0000-0000-000000000000"
		if index > 0 {
			id = "10000000-0000-0000-0000-" + formatFixedIDSuffix(index)
		}
		index++
		return id
	}
}

func formatFixedIDSuffix(index int) string {
	return fmt.Sprintf("%012d", index)
}

func doJSONRequest(t *testing.T, h http.Handler, method, path, token string, body map[string]any) *httptest.ResponseRecorder {
	t.Helper()
	buf, err := json.Marshal(body)
	if err != nil {
		t.Fatalf("marshal request: %v", err)
	}
	req := httptest.NewRequest(method, path, bytes.NewReader(buf))
	req.Header.Set("Content-Type", "application/json")
	if token != "" {
		req.Header.Set("Authorization", "Bearer "+token)
	}
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, req)
	return rec
}

func decodeRecorder(t *testing.T, rec *httptest.ResponseRecorder, out any) {
	t.Helper()
	if err := json.NewDecoder(rec.Body).Decode(out); err != nil {
		t.Fatalf("decode recorder: %v", err)
	}
}
