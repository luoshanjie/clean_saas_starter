//go:build integration

package integration

import (
	"context"
	"database/sql"
	"net/http"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/labstack/echo/v4"

	"service/internal/app/usecase"
	"service/internal/bootstrap"
	"service/internal/delivery/http/handler"
	"service/internal/delivery/http/middleware"
	"service/internal/domain/model"
	sqliterepo "service/internal/repo/sqlite"
	storagerepo "service/internal/repo/storage"
	"service/pkg/logger"
)

type directLoginResp struct {
	Code    int    `json:"code"`
	Message string `json:"message"`
	Data    struct {
		RequiresSecondFactor bool   `json:"requires_second_factor"`
		AccessToken          string `json:"access_token"`
		RefreshToken         string `json:"refresh_token"`
	} `json:"data"`
}

type fileUploadSessionCreateResp struct {
	Code    int    `json:"code"`
	Message string `json:"message"`
	Data    struct {
		UploadID  string `json:"upload_id"`
		UploadURL string `json:"upload_url"`
		FileURL   string `json:"file_url"`
		Status    string `json:"status"`
	} `json:"data"`
}

type fileUploadConfirmResp struct {
	Code    int    `json:"code"`
	Message string `json:"message"`
	Data    struct {
		Status  string `json:"status"`
		FileID  string `json:"file_id"`
		FileURL string `json:"file_url"`
	} `json:"data"`
}

type fileDownloadPresignResp struct {
	Code    int    `json:"code"`
	Message string `json:"message"`
	Data    struct {
		DownloadURL string `json:"download_url"`
	} `json:"data"`
}

type fileDeleteResp struct {
	Code    int    `json:"code"`
	Message string `json:"message"`
	Data    struct {
		Status string `json:"status"`
	} `json:"data"`
}

func TestSQLiteFileFlow_Integration(t *testing.T) {
	chdirToRepoRoot(t)

	ctx := context.Background()
	dbPath := filepath.Join(t.TempDir(), "file-integration.db")
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

	now := func() time.Time { return time.Date(2026, 3, 8, 12, 0, 0, 0, time.UTC) }
	e := newSQLiteFileTestApp(t, runtime.SQLite, now)

	login := doJSONRequest(t, e, http.MethodPost, "/api/v1/auth/login", "", map[string]any{
		"account":  "platform_root",
		"password": "pass123",
	})
	var lr directLoginResp
	decodeRecorder(t, login, &lr)
	if lr.Code != 0 || lr.Data.AccessToken == "" || lr.Data.RequiresSecondFactor {
		t.Fatalf("direct login failed: %+v", lr)
	}

	create := doJSONRequest(t, e, http.MethodPost, "/api/v1/file/upload/session/create", lr.Data.AccessToken, map[string]any{
		"biz_type":     "attachment",
		"file_name":    "contract.pdf",
		"content_type": "application/pdf",
		"size_bytes":   2048,
	})
	var cr fileUploadSessionCreateResp
	decodeRecorder(t, create, &cr)
	if cr.Code != 0 || cr.Data.UploadID == "" || cr.Data.UploadURL == "" || cr.Data.FileURL == "" {
		t.Fatalf("upload session create failed: %+v", cr)
	}
	if !strings.Contains(cr.Data.UploadURL, "mock-storage.local/upload/attachment/contract.pdf") {
		t.Fatalf("unexpected upload url: %+v", cr)
	}

	confirm := doJSONRequest(t, e, http.MethodPost, "/api/v1/file/upload/confirm", lr.Data.AccessToken, map[string]any{
		"upload_id": cr.Data.UploadID,
	})
	var cfr fileUploadConfirmResp
	decodeRecorder(t, confirm, &cfr)
	if cfr.Code != 0 || cfr.Data.Status != "ok" || cfr.Data.FileID == "" {
		t.Fatalf("upload confirm failed: %+v", cfr)
	}
	if cfr.Data.FileID != cr.Data.UploadID {
		t.Fatalf("expected file id to equal upload id, got %+v", cfr)
	}

	assertSQLiteFileAsset(t, runtime.SQLite, cfr.Data.FileID)

	download := doJSONRequest(t, e, http.MethodPost, "/api/v1/file/download/presign", lr.Data.AccessToken, map[string]any{
		"file_id":  cfr.Data.FileID,
		"file_url": "https://should-not-be-used.invalid/file.pdf",
	})
	var dr fileDownloadPresignResp
	decodeRecorder(t, download, &dr)
	if dr.Code != 0 || dr.Data.DownloadURL == "" {
		t.Fatalf("download presign failed: %+v", dr)
	}
	if !strings.Contains(dr.Data.DownloadURL, "mock-storage.local/download/attachment/contract.pdf") {
		t.Fatalf("unexpected download url: %+v", dr)
	}
	if strings.Contains(dr.Data.DownloadURL, "should-not-be-used.invalid") {
		t.Fatalf("expected file_id to take precedence over file_url, got %+v", dr)
	}

	del := doJSONRequest(t, e, http.MethodPost, "/api/v1/file/delete", lr.Data.AccessToken, map[string]any{
		"file_id": cfr.Data.FileID,
	})
	var drr fileDeleteResp
	decodeRecorder(t, del, &drr)
	if drr.Code != 0 || drr.Data.Status != "ok" {
		t.Fatalf("file delete failed: %+v", drr)
	}
	assertSQLiteFileAssetMissing(t, runtime.SQLite, cfr.Data.FileID)

	downloadAfterDelete := doJSONRequest(t, e, http.MethodPost, "/api/v1/file/download/presign", lr.Data.AccessToken, map[string]any{
		"file_id": cfr.Data.FileID,
	})
	decodeRecorder(t, downloadAfterDelete, &dr)
	if dr.Code == 0 {
		t.Fatalf("expected deleted file to become unavailable, got %+v", dr)
	}
}

func newSQLiteFileTestApp(t *testing.T, db *sql.DB, now func() time.Time) *echo.Echo {
	t.Helper()
	authRepo := &sqliterepo.AuthRepoSQLite{DB: db}
	fileUploadRepo := &sqliterepo.FileUploadSessionRepoSQLite{DB: db}
	fileRepo := &sqliterepo.FileRepoSQLite{DB: db}
	storage := &storagerepo.MockObjectStorage{Now: now}

	tokenGen := func(ctx context.Context, user *model.User) (string, string, int, error) {
		accessExpireSec := 3600
		refreshExpireSec := 604800
		nowAt := now()
		refresh := middleware.Claims{
			UserID:       user.ID,
			Name:         user.Name,
			Role:         user.Role,
			ScopeType:    user.ScopeType,
			TenantID:     user.TenantID,
			TokenType:    "refresh",
			TokenVersion: user.TokenVersion,
			RegisteredClaims: jwt.RegisteredClaims{
				ExpiresAt: jwt.NewNumericDate(nowAt.Add(time.Duration(refreshExpireSec) * time.Second)),
			},
		}
		claims := middleware.Claims{
			UserID:       user.ID,
			Name:         user.Name,
			Role:         user.Role,
			ScopeType:    user.ScopeType,
			TenantID:     user.TenantID,
			TokenType:    "access",
			TokenVersion: user.TokenVersion,
			RegisteredClaims: jwt.RegisteredClaims{
				ExpiresAt: jwt.NewNumericDate(nowAt.Add(time.Duration(accessExpireSec) * time.Second)),
			},
		}
		accessToken := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
		refreshToken := jwt.NewWithClaims(jwt.SigningMethodHS256, refresh)
		a, err := accessToken.SignedString([]byte("test_secret"))
		if err != nil {
			return "", "", 0, err
		}
		r, err := refreshToken.SignedString([]byte("test_secret"))
		if err != nil {
			return "", "", 0, err
		}
		return a, r, accessExpireSec, nil
	}

	authHandler := &handler.AuthHandler{
		LoginUC: &usecase.AuthLoginUsecase{
			Repo:     authRepo,
			TokenGen: tokenGen,
		},
		JWT:                      middleware.JWTMiddleware{Secret: []byte("test_secret")},
		LoginSecondFactorEnabled: false,
	}
	fileHandler := &handler.FileHandler{
		UploadSessionCreateUC: &usecase.FileUploadSessionCreateUsecase{
			Storage: storage,
			Repo:    fileUploadRepo,
			IDGen: func() string {
				return "30000000-0000-0000-0000-000000000001"
			},
			Now: now,
		},
		UploadConfirmUC: &usecase.FileUploadConfirmUsecase{
			Repo:     fileUploadRepo,
			FileRepo: fileRepo,
			Storage:  storage,
			Now:      now,
		},
		DownloadPresignUC: &usecase.FileDownloadPresignUsecase{
			Storage:  storage,
			FileRepo: fileRepo,
		},
		DeleteUC: &usecase.FileDeleteUsecase{
			Storage:  storage,
			FileRepo: fileRepo,
		},
	}

	e := bootstrap.NewEcho(logger.NewNopLogger())
	api := e.Group("/api/v1")
	api.POST("/auth/login", authHandler.Login)
	protected := api.Group("", authHandler.JWT.MiddlewareFunc, middleware.AuthContextMiddleware(authRepo))
	protected.POST("/file/upload/session/create", fileHandler.UploadSessionCreate)
	protected.POST("/file/upload/confirm", fileHandler.UploadConfirm)
	protected.POST("/file/download/presign", fileHandler.DownloadPresign)
	protected.POST("/file/delete", fileHandler.Delete)
	return e
}

func assertSQLiteFileAsset(t *testing.T, db *sql.DB, fileID string) {
	t.Helper()
	var (
		tenantID  string
		bucket    string
		objectKey string
		ownerType string
		ownerID   string
	)
	err := db.QueryRow(`
SELECT COALESCE(tenant_id, ''), bucket, object_key, owner_type, owner_id
FROM files
WHERE id = ?`, fileID).Scan(&tenantID, &bucket, &objectKey, &ownerType, &ownerID)
	if err != nil {
		t.Fatalf("query file asset: %v", err)
	}
	if tenantID != "" {
		t.Fatalf("expected platform file to have empty tenant id, got %s", tenantID)
	}
	if bucket != "mock" {
		t.Fatalf("unexpected bucket: %s", bucket)
	}
	if objectKey != "attachment/contract.pdf" {
		t.Fatalf("unexpected object key: %s", objectKey)
	}
	if ownerType != "user" || ownerID != "00000000-0000-0000-0000-000000000001" {
		t.Fatalf("unexpected owner binding: owner_type=%s owner_id=%s", ownerType, ownerID)
	}
}

func assertSQLiteFileAssetMissing(t *testing.T, db *sql.DB, fileID string) {
	t.Helper()
	var count int
	if err := db.QueryRow(`SELECT COUNT(1) FROM files WHERE id = ?`, fileID).Scan(&count); err != nil {
		t.Fatalf("query deleted file asset: %v", err)
	}
	if count != 0 {
		t.Fatalf("expected file asset to be deleted, count=%d", count)
	}
}
