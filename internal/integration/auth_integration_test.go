//go:build integration

package integration

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"runtime"
	"testing"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/labstack/echo/v4"
	"golang.org/x/crypto/bcrypt"

	"service/internal/bootstrap"
	"service/internal/domain/authctx"
	"service/internal/repo/pg"
	"service/pkg/logger"
)

type loginChallengeResp struct {
	Code    int    `json:"code"`
	Message string `json:"message"`
	Data    struct {
		ChallengeID string `json:"challenge_id"`
		MaskedPhone string `json:"masked_phone"`
	} `json:"data"`
}

type loginVerifyResp struct {
	Code    int    `json:"code"`
	Message string `json:"message"`
	Data    struct {
		AccessToken  string `json:"access_token"`
		RefreshToken string `json:"refresh_token"`
	} `json:"data"`
}

type meResp struct {
	Code int `json:"code"`
	Data struct {
		User struct {
			ID string `json:"id"`
		} `json:"user"`
		Permissions []string `json:"permissions"`
	} `json:"data"`
}

type refreshResp struct {
	Code int `json:"code"`
	Data struct {
		AccessToken  string `json:"access_token"`
		RefreshToken string `json:"refresh_token"`
	} `json:"data"`
}

func TestAuthFlow_Integration(t *testing.T) {
	chdirToRepoRoot(t)
	dsn := os.Getenv("INTEGRATION_DB_DSN")
	if dsn == "" {
		dsn = os.Getenv("DB_DSN")
	}
	if dsn == "" {
		t.Skip("INTEGRATION_DB_DSN/DB_DSN not set")
	}

	ctx := context.Background()
	pool, err := pgxpool.New(ctx, dsn)
	if err != nil {
		t.Fatalf("pgxpool: %v", err)
	}
	defer pool.Close()

	cleanup(t, pool)
	seed(t, pool)
	assertSeed(t, pool)
	assertLoginQuery(t, pool)
	assertAuthRepo(t, pool)

	e := echo.New()
	_, err = bootstrap.Build(ctx, e, pool, func() string { return "tid" }, time.Now, bootstrap.Config{JWTSecret: "test_secret"}, logger.NewNopLogger())
	if err != nil {
		t.Fatalf("build: %v", err)
	}
	srv := httptest.NewServer(e)
	defer srv.Close()

	// login
	login := postJSON(t, srv.URL+"/api/v1/auth/login", map[string]any{
		"account":  "member1",
		"password": "pass123",
	})
	var lr loginChallengeResp
	decode(t, login, &lr)
	if lr.Code != 0 {
		t.Logf("login error msg: %s", lr.Message)
		t.Fatalf("login failed: %+v", lr)
	}
	if lr.Data.ChallengeID == "" {
		t.Fatalf("login failed: %+v", lr)
	}
	verify := postJSON(t, srv.URL+"/api/v1/auth/login/verify", map[string]any{
		"challenge_id": lr.Data.ChallengeID,
		"otp_code":     "123456",
	})
	var lv loginVerifyResp
	decode(t, verify, &lv)
	if lv.Code != 0 {
		t.Fatalf("verify failed: %+v", lv)
	}

	// me with access token
	me := postAuthJSON(t, srv.URL+"/api/v1/auth/me", lv.Data.AccessToken, map[string]any{})
	var mr meResp
	decode(t, me, &mr)
	if mr.Code != 0 || mr.Data.User.ID == "" {
		t.Fatalf("me failed: %+v", mr)
	}

	// refresh
	rf := postJSON(t, srv.URL+"/api/v1/auth/refresh", map[string]any{
		"refresh_token": lv.Data.RefreshToken,
	})
	var rr refreshResp
	decode(t, rf, &rr)
	if rr.Code != 0 || rr.Data.AccessToken == "" || rr.Data.RefreshToken == "" {
		t.Fatalf("refresh failed: %+v", rr)
	}

	// refresh token should be rejected by auth middleware
	me2 := postAuthJSON(t, srv.URL+"/api/v1/auth/me", lv.Data.RefreshToken, map[string]any{})
	var mr2 meResp
	decode(t, me2, &mr2)
	if mr2.Code == 0 {
		t.Fatalf("refresh token should not access /api/v1/auth/me")
	}

	// token_version mismatch after update
	bumpTokenVersion(t, pool)
	me3 := postAuthJSON(t, srv.URL+"/api/v1/auth/me", lv.Data.AccessToken, map[string]any{})
	var mr3 meResp
	decode(t, me3, &mr3)
	if mr3.Code == 0 {
		t.Fatalf("token version mismatch should be unauthorized")
	}
}

func chdirToRepoRoot(t *testing.T) {
	_, file, _, ok := runtime.Caller(0)
	if !ok {
		t.Fatalf("cannot get caller")
	}
	root := filepath.Dir(filepath.Dir(filepath.Dir(file)))
	if err := os.Chdir(root); err != nil {
		t.Fatalf("chdir: %v", err)
	}
}

func cleanup(t *testing.T, pool *pgxpool.Pool) {
	t.Helper()
	tx, err := pool.Begin(context.Background())
	if err != nil {
		t.Fatalf("begin: %v", err)
	}
	defer tx.Rollback(context.Background())
	if _, err := tx.Exec(context.Background(), "SET LOCAL app.scope_type = 'platform'"); err != nil {
		t.Fatalf("set scope: %v", err)
	}
	_, _ = tx.Exec(context.Background(), "TRUNCATE login_challenges, user_credentials, users, tenants, rbac_policies RESTART IDENTITY CASCADE")
	if err := tx.Commit(context.Background()); err != nil {
		t.Fatalf("commit: %v", err)
	}
}

func seed(t *testing.T, pool *pgxpool.Pool) {
	t.Helper()
	tenantID := "00000000-0000-0000-0000-000000000001"
	userID := "00000000-0000-0000-0000-000000000101"

	hash, _ := bcrypt.GenerateFromPassword([]byte("pass123"), bcrypt.DefaultCost)

	tx, err := pool.Begin(context.Background())
	if err != nil {
		t.Fatalf("begin: %v", err)
	}
	defer tx.Rollback(context.Background())
	if _, err := tx.Exec(context.Background(), "SET LOCAL app.scope_type = 'platform'"); err != nil {
		t.Fatalf("set scope: %v", err)
	}

	var c int
	if err := tx.QueryRow(context.Background(), "SELECT count(*) FROM users WHERE id=$1", userID).Scan(&c); err != nil {
		t.Fatalf("count users: %v", err)
	}
	if c != 0 {
		t.Fatalf("unexpected user exists before insert")
	}

	_, err = tx.Exec(context.Background(), `INSERT INTO tenants (id, name) VALUES ($1, $2)`, tenantID, "t1")
	if err != nil {
		t.Fatalf("insert tenant: %v", err)
	}
	_, err = tx.Exec(context.Background(), `
		INSERT INTO users (id, tenant_id, name, phone, role, scope_type, status, token_version)
		VALUES ($1,$2,$3,$4,$5,$6,'active',0)
	`, userID, tenantID, "member1", "13800000003", "tenant_member", "tenant")
	if err != nil {
		t.Fatalf("insert user: %v", err)
	}
	_, err = tx.Exec(context.Background(), `
		INSERT INTO user_credentials (user_id, account, password_hash)
		VALUES ($1,$2,$3)
	`, userID, "member1", string(hash))
	if err != nil {
		t.Fatalf("insert credentials: %v", err)
	}
	if err := tx.Commit(context.Background()); err != nil {
		t.Fatalf("commit: %v", err)
	}
}

func assertSeed(t *testing.T, pool *pgxpool.Pool) {
	t.Helper()
	tx, err := pool.Begin(context.Background())
	if err != nil {
		t.Fatalf("begin: %v", err)
	}
	defer tx.Rollback(context.Background())
	if _, err := tx.Exec(context.Background(), "SET LOCAL app.scope_type = 'platform'"); err != nil {
		t.Fatalf("set scope: %v", err)
	}
	var cnt int
	if err := tx.QueryRow(context.Background(), "SELECT count(*) FROM user_credentials").Scan(&cnt); err != nil {
		t.Fatalf("count credentials: %v", err)
	}
	if cnt != 1 {
		t.Fatalf("expected 1 credential, got %d", cnt)
	}
	var acc string
	var status string
	if err := tx.QueryRow(context.Background(), `
		SELECT c.account, u.status
		FROM user_credentials c
		JOIN users u ON u.id = c.user_id
		LIMIT 1
	`).Scan(&acc, &status); err != nil {
		t.Fatalf("query account: %v", err)
	}
	if acc != "member1" || status != "active" {
		t.Fatalf("unexpected seed data: account=%s status=%s", acc, status)
	}
}

func assertLoginQuery(t *testing.T, pool *pgxpool.Pool) {
	t.Helper()
	tx, err := pool.Begin(context.Background())
	if err != nil {
		t.Fatalf("begin: %v", err)
	}
	defer tx.Rollback(context.Background())
	if _, err := tx.Exec(context.Background(), "SET LOCAL app.scope_type = 'platform'"); err != nil {
		t.Fatalf("set scope: %v", err)
	}
	var hash string
	if err := tx.QueryRow(context.Background(), `
		SELECT c.password_hash
		FROM user_credentials c
		WHERE c.account = $1
	`, "member1").Scan(&hash); err != nil {
		t.Fatalf("query credentials: %v", err)
	}
	if err := bcrypt.CompareHashAndPassword([]byte(hash), []byte("pass123")); err != nil {
		t.Fatalf("bcrypt compare failed: %v", err)
	}
}

func assertAuthRepo(t *testing.T, pool *pgxpool.Pool) {
	t.Helper()
	repo := &pg.AuthRepoPG{DB: pool}
	ctx := authctx.With(context.Background(), authctx.Info{ScopeType: "platform"})
	if _, _, err := repo.GetUserByAccount(ctx, "member1"); err != nil {
		t.Fatalf("auth repo query failed: %v", err)
	}
}

func bumpTokenVersion(t *testing.T, pool *pgxpool.Pool) {
	t.Helper()
	tx, err := pool.Begin(context.Background())
	if err != nil {
		t.Fatalf("begin: %v", err)
	}
	defer tx.Rollback(context.Background())
	if _, err := tx.Exec(context.Background(), "SET LOCAL app.scope_type = 'platform'"); err != nil {
		t.Fatalf("set scope: %v", err)
	}
	_, err = tx.Exec(context.Background(), `UPDATE users SET token_version = token_version + 1`)
	if err != nil {
		t.Fatalf("update token_version: %v", err)
	}
	if err := tx.Commit(context.Background()); err != nil {
		t.Fatalf("commit: %v", err)
	}
}

func postJSON(t *testing.T, url string, body map[string]any) *http.Response {
	t.Helper()
	buf, _ := json.Marshal(body)
	resp, err := http.Post(url, "application/json", bytes.NewReader(buf))
	if err != nil {
		t.Fatalf("post: %v", err)
	}
	return resp
}

func postAuthJSON(t *testing.T, url, token string, body map[string]any) *http.Response {
	t.Helper()
	buf, _ := json.Marshal(body)
	req, _ := http.NewRequest(http.MethodPost, url, bytes.NewReader(buf))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+token)
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatalf("post auth: %v", err)
	}
	return resp
}

func decode(t *testing.T, resp *http.Response, out any) {
	t.Helper()
	defer resp.Body.Close()
	if err := json.NewDecoder(resp.Body).Decode(out); err != nil {
		t.Fatalf("decode: %v", err)
	}
}
