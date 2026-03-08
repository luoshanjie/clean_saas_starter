package bootstrap

import (
	"os"
	"path/filepath"
	"testing"
	"time"
)

func TestOSSConfigEnabled(t *testing.T) {
	t.Run("disabled_when_required_fields_missing", func(t *testing.T) {
		cfg := OSSConfig{
			Endpoint:  "127.0.0.1:9000",
			AccessKey: "minioadmin",
			SecretKey: "",
			Bucket:    "service-dev",
		}
		if cfg.Enabled() {
			t.Fatalf("expected OSS disabled when secret key is missing")
		}
	})

	t.Run("enabled_when_required_fields_present", func(t *testing.T) {
		cfg := OSSConfig{
			Endpoint:  "127.0.0.1:9000",
			AccessKey: "minioadmin",
			SecretKey: "miniosecret",
			Bucket:    "service-dev",
		}
		if !cfg.Enabled() {
			t.Fatalf("expected OSS enabled when required fields are present")
		}
	})
}

func TestNormalizeDBDriver(t *testing.T) {
	cases := map[string]string{
		"":          DBDriverPostgres,
		"postgres":  DBDriverPostgres,
		"POSTGRES":  DBDriverPostgres,
		"sqlite":    DBDriverSQLite,
		"sqlite3":   DBDriverSQLite,
		"SQLITE3":   DBDriverSQLite,
		"custom-db": "custom-db",
	}
	for in, want := range cases {
		if got := normalizeDBDriver(in); got != want {
			t.Fatalf("normalizeDBDriver(%q) = %q, want %q", in, got, want)
		}
	}
}

func TestLoadConfig_FromYAML(t *testing.T) {
	clearConfigEnv(t)
	dir := t.TempDir()
	configFile := filepath.Join(dir, "app.yaml")
	err := os.WriteFile(configFile, []byte(`
server:
  addr: ":9527"
  jwt_secret: "yaml_secret"
database:
  driver: "sqlite3"
  dsn: "postgres://u:p@10.0.0.1:5432/service?sslmode=disable"
  sqlite_path: "./var/service.db"
auth:
  login_second_factor_enabled: true
oss:
  endpoint: "127.0.0.1:9000"
  access_key: "minioadmin"
  secret_key: "oss_key"
  bucket: "bucket-123"
  public_base_url: "https://storage.example.com/bucket-123"
  use_ssl: false
log:
  dir: "/app/logs"
  level: "info"
  format: "json"
  console_format: "json"
  console: false
`), 0o644)
	if err != nil {
		t.Fatalf("write config file: %v", err)
	}
	t.Setenv("APP_CONFIG_FILE", configFile)

	cfg, err := LoadConfig()
	if err != nil {
		t.Fatalf("LoadConfig error: %v", err)
	}
	if cfg.Addr != ":9527" {
		t.Fatalf("unexpected addr: %s", cfg.Addr)
	}
	if cfg.JWTSecret != "yaml_secret" {
		t.Fatalf("unexpected JWT secret: %s", cfg.JWTSecret)
	}
	if cfg.DBDSN != "postgres://u:p@10.0.0.1:5432/service?sslmode=disable" {
		t.Fatalf("unexpected dsn: %s", cfg.DBDSN)
	}
	if cfg.DBDriver != DBDriverSQLite {
		t.Fatalf("unexpected db driver: %s", cfg.DBDriver)
	}
	if cfg.SQLitePath != "./var/service.db" {
		t.Fatalf("unexpected sqlite path: %s", cfg.SQLitePath)
	}
	if !cfg.Auth.LoginSecondFactorEnabled {
		t.Fatalf("expected auth login second factor enabled from yaml")
	}
	if cfg.OSS.Endpoint != "127.0.0.1:9000" {
		t.Fatalf("unexpected oss endpoint: %s", cfg.OSS.Endpoint)
	}
	if cfg.OSS.AccessKey != "minioadmin" {
		t.Fatalf("unexpected oss access key: %s", cfg.OSS.AccessKey)
	}
	if cfg.Log.EnableConsole {
		t.Fatalf("unexpected log console: true")
	}
}

func TestLoadConfig_EnvOverridesYAML(t *testing.T) {
	clearConfigEnv(t)
	dir := t.TempDir()
	configFile := filepath.Join(dir, "app.yaml")
	err := os.WriteFile(configFile, []byte(`
server:
  addr: ":9527"
  jwt_secret: "yaml_secret"
database:
  dsn: "postgres://u:p@10.0.0.1:5432/service?sslmode=disable"
log:
  dir: "/app/logs"
  level: "info"
  format: "json"
  console_format: "json"
  console: false
`), 0o644)
	if err != nil {
		t.Fatalf("write config file: %v", err)
	}

	t.Setenv("APP_CONFIG_FILE", configFile)
	t.Setenv("ADDR", ":9000")
	t.Setenv("DB_DRIVER", "postgres")
	t.Setenv("DB_DSN", "postgres://env:env@10.0.0.2:5432/service?sslmode=disable")
	t.Setenv("SQLITE_PATH", "./var/env.db")
	t.Setenv("JWT_SECRET", "env_secret")
	t.Setenv("AUTH_LOGIN_SECOND_FACTOR_ENABLED", "true")
	t.Setenv("LOG_LEVEL", "debug")
	t.Setenv("LOG_CONSOLE", "1")
	t.Setenv("OSS_ACCESS_KEY", "env_minio_access_key")

	cfg, err := LoadConfig()
	if err != nil {
		t.Fatalf("LoadConfig error: %v", err)
	}
	if cfg.Addr != ":9000" {
		t.Fatalf("unexpected addr: %s", cfg.Addr)
	}
	if cfg.JWTSecret != "env_secret" {
		t.Fatalf("unexpected JWT secret: %s", cfg.JWTSecret)
	}
	if cfg.DBDSN != "postgres://env:env@10.0.0.2:5432/service?sslmode=disable" {
		t.Fatalf("unexpected dsn: %s", cfg.DBDSN)
	}
	if cfg.DBDriver != DBDriverPostgres {
		t.Fatalf("unexpected db driver: %s", cfg.DBDriver)
	}
	if cfg.SQLitePath != "./var/env.db" {
		t.Fatalf("unexpected sqlite path: %s", cfg.SQLitePath)
	}
	if !cfg.Auth.LoginSecondFactorEnabled {
		t.Fatalf("expected auth login second factor enabled from env")
	}
	if cfg.Log.Level != "debug" {
		t.Fatalf("unexpected log level: %s", cfg.Log.Level)
	}
	if cfg.OSS.AccessKey != "env_minio_access_key" {
		t.Fatalf("unexpected oss access key: %s", cfg.OSS.AccessKey)
	}
	if !cfg.Log.EnableConsole {
		t.Fatalf("unexpected log console: false")
	}
}

func TestLoadConfig_EnvOnly(t *testing.T) {
	clearConfigEnv(t)
	t.Setenv("APP_CONFIG_FILE", filepath.Join(t.TempDir(), "not-found.yaml"))
	t.Setenv("ADDR", ":9527")
	t.Setenv("DB_DRIVER", "sqlite")
	t.Setenv("DB_DSN", "postgres://env:env@10.0.0.2:5432/service?sslmode=disable")
	t.Setenv("SQLITE_PATH", "./var/local.db")
	t.Setenv("JWT_SECRET", "env_secret")
	t.Setenv("LOG_DIR", "/tmp/logs")
	t.Setenv("LOG_CONSOLE", "0")

	_, err := LoadConfig()
	if err == nil {
		t.Fatalf("expected error when APP_CONFIG_FILE does not exist")
	}

	t.Setenv("APP_CONFIG_FILE", "")
	cfg, err := LoadConfig()
	if err != nil {
		t.Fatalf("LoadConfig error: %v", err)
	}
	if cfg.Addr != ":9527" {
		t.Fatalf("unexpected addr: %s", cfg.Addr)
	}
	if cfg.JWTSecret != "env_secret" {
		t.Fatalf("unexpected JWT secret: %s", cfg.JWTSecret)
	}
	if cfg.DBDSN != "postgres://env:env@10.0.0.2:5432/service?sslmode=disable" {
		t.Fatalf("unexpected dsn: %s", cfg.DBDSN)
	}
	if cfg.DBDriver != DBDriverSQLite {
		t.Fatalf("unexpected db driver: %s", cfg.DBDriver)
	}
	if cfg.SQLitePath != "./var/local.db" {
		t.Fatalf("unexpected sqlite path: %s", cfg.SQLitePath)
	}
	if cfg.Auth.LoginSecondFactorEnabled {
		t.Fatalf("expected auth login second factor disabled by default")
	}
	if cfg.Log.Dir != "/tmp/logs" {
		t.Fatalf("unexpected log dir: %s", cfg.Log.Dir)
	}
	if cfg.Log.EnableConsole {
		t.Fatalf("unexpected log console: true")
	}
}

func TestLoadConfig_UploadCleanupEnvOverrides(t *testing.T) {
	clearConfigEnv(t)
	t.Setenv("APP_CONFIG_FILE", "")
	t.Setenv("ADDR", ":9527")
	t.Setenv("DB_DSN", "postgres://env:env@10.0.0.2:5432/service?sslmode=disable")
	t.Setenv("JWT_SECRET", "env_secret")
	t.Setenv("UPLOAD_CLEANUP_ENABLED", "1")
	t.Setenv("UPLOAD_CLEANUP_INTERVAL_SEC", "60")
	t.Setenv("UPLOAD_CLEANUP_BATCH_SIZE", "500")

	cfg, err := LoadConfig()
	if err != nil {
		t.Fatalf("LoadConfig error: %v", err)
	}
	if !cfg.UploadCleanup.Enabled {
		t.Fatalf("expected upload cleanup enabled")
	}
	if cfg.UploadCleanup.Interval != 60*time.Second {
		t.Fatalf("unexpected upload cleanup interval: %v", cfg.UploadCleanup.Interval)
	}
	if cfg.UploadCleanup.BatchSize != 500 {
		t.Fatalf("unexpected upload cleanup batch size: %d", cfg.UploadCleanup.BatchSize)
	}
}

func clearConfigEnv(t *testing.T) {
	t.Helper()
	keys := []string{
		"APP_CONFIG_FILE",
		"ADDR",
		"DB_DRIVER",
		"DB_DSN",
		"SQLITE_PATH",
		"SKIP_DB",
		"JWT_SECRET",
		"APP_ENV",
		"OSS_ENDPOINT",
		"OSS_ACCESS_KEY",
		"OSS_SECRET_KEY",
		"OSS_BUCKET",
		"OSS_PUBLIC_BASE_URL",
		"OSS_USE_SSL",
		"LOG_DIR",
		"LOG_LEVEL",
		"LOG_FORMAT",
		"LOG_CONSOLE_FORMAT",
		"LOG_CONSOLE",
		"UPLOAD_CLEANUP_ENABLED",
		"UPLOAD_CLEANUP_INTERVAL_SEC",
		"UPLOAD_CLEANUP_BATCH_SIZE",
	}
	for _, k := range keys {
		t.Setenv(k, "")
	}
}
