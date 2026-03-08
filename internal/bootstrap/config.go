package bootstrap

import (
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"gopkg.in/yaml.v2"

	"service/pkg/logger"
)

// Config 只存放启动所需的最小配置。
type Config struct {
	Addr          string
	DBDriver      string
	DBDSN         string
	SQLitePath    string
	SkipDB        bool
	JWTSecret     string
	OSS           OSSConfig
	UploadCleanup UploadCleanupConfig
	Log           logger.Config
}

const (
	DBDriverPostgres = "postgres"
	DBDriverSQLite   = "sqlite"
)

type UploadCleanupConfig struct {
	Enabled   bool
	Interval  time.Duration
	BatchSize int
}

type OSSConfig struct {
	Endpoint      string
	AccessKey     string
	SecretKey     string
	Bucket        string
	PublicBaseURL string
	UseSSL        bool
}

func (c OSSConfig) Enabled() bool {
	return strings.TrimSpace(c.Endpoint) != "" &&
		strings.TrimSpace(c.AccessKey) != "" &&
		strings.TrimSpace(c.SecretKey) != "" &&
		strings.TrimSpace(c.Bucket) != ""
}

type fileConfig struct {
	Server struct {
		Addr      string `yaml:"addr"`
		JWTSecret string `yaml:"jwt_secret"`
	} `yaml:"server"`
	Database struct {
		Driver     string `yaml:"driver"`
		DSN        string `yaml:"dsn"`
		SQLitePath string `yaml:"sqlite_path"`
	} `yaml:"database"`
	OSS struct {
		Endpoint      string `yaml:"endpoint"`
		AccessKey     string `yaml:"access_key"`
		SecretKey     string `yaml:"secret_key"`
		Bucket        string `yaml:"bucket"`
		PublicBaseURL string `yaml:"public_base_url"`
		UseSSL        *bool  `yaml:"use_ssl"`
	} `yaml:"oss"`
	UploadCleanup struct {
		Enabled     *bool `yaml:"enabled"`
		IntervalSec int   `yaml:"interval_sec"`
		BatchSize   int   `yaml:"batch_size"`
	} `yaml:"upload_cleanup"`
	Log struct {
		Dir           string `yaml:"dir"`
		Level         string `yaml:"level"`
		Format        string `yaml:"format"`
		ConsoleFormat string `yaml:"console_format"`
		Console       *bool  `yaml:"console"`
	} `yaml:"log"`
}

func LoadConfig() (Config, error) {
	appEnv := os.Getenv("APP_ENV")
	defaultConsoleFormat := "json"
	if strings.EqualFold(appEnv, "dev") {
		defaultConsoleFormat = "text"
	}

	cfg := Config{
		Addr:       getenvDefault("ADDR", ":8080"),
		DBDriver:   normalizeDBDriver(getenvDefault("DB_DRIVER", DBDriverPostgres)),
		DBDSN:      os.Getenv("DB_DSN"),
		SQLitePath: getenvDefault("SQLITE_PATH", ""),
		SkipDB:     os.Getenv("SKIP_DB") == "1",
		JWTSecret:  os.Getenv("JWT_SECRET"),
		Log: logger.Config{
			Dir:           getenvDefault("LOG_DIR", "./logs"),
			Level:         getenvDefault("LOG_LEVEL", "info"),
			Format:        getenvDefault("LOG_FORMAT", "json"),
			ConsoleFormat: getenvDefault("LOG_CONSOLE_FORMAT", defaultConsoleFormat),
			EnableConsole: !strings.EqualFold(os.Getenv("LOG_CONSOLE"), "0"),
		},
		OSS: OSSConfig{
			Endpoint:      getenvDefault("OSS_ENDPOINT", ""),
			AccessKey:     getenvDefault("OSS_ACCESS_KEY", ""),
			SecretKey:     getenvDefault("OSS_SECRET_KEY", ""),
			Bucket:        getenvDefault("OSS_BUCKET", ""),
			PublicBaseURL: getenvDefault("OSS_PUBLIC_BASE_URL", ""),
			UseSSL:        parseBoolEnv("OSS_USE_SSL", true),
		},
		UploadCleanup: UploadCleanupConfig{
			Enabled:   true,
			Interval:  10 * time.Minute,
			BatchSize: 200,
		},
	}

	if p, ok := resolveConfigFilePath(); ok {
		fileCfg, err := loadConfigFile(p)
		if err != nil {
			return Config{}, err
		}
		mergeFileConfig(&cfg, fileCfg)
	}
	applyEnvOverrides(&cfg, defaultConsoleFormat)
	return cfg, nil
}

func getenvDefault(key, def string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return def
}

func parseBoolEnv(key string, def bool) bool {
	v := strings.TrimSpace(os.Getenv(key))
	if v == "" {
		return def
	}
	return !strings.EqualFold(v, "0") && !strings.EqualFold(v, "false")
}

func resolveConfigFilePath() (string, bool) {
	if p := strings.TrimSpace(os.Getenv("APP_CONFIG_FILE")); p != "" {
		return p, true
	}
	candidates := []string{"/app/config/app.yaml", "./app.yaml"}
	for _, p := range candidates {
		if _, err := os.Stat(p); err == nil {
			return p, true
		}
	}
	return "", false
}

func loadConfigFile(path string) (fileConfig, error) {
	content, err := os.ReadFile(filepath.Clean(path))
	if err != nil {
		return fileConfig{}, fmt.Errorf("read config file %s: %w", path, err)
	}
	var cfg fileConfig
	if err := yaml.Unmarshal(content, &cfg); err != nil {
		return fileConfig{}, fmt.Errorf("parse config file %s: %w", path, err)
	}
	return cfg, nil
}

func mergeFileConfig(dst *Config, src fileConfig) {
	if src.Server.Addr != "" {
		dst.Addr = src.Server.Addr
	}
	if src.Server.JWTSecret != "" {
		dst.JWTSecret = src.Server.JWTSecret
	}
	if src.Database.DSN != "" {
		dst.DBDSN = src.Database.DSN
	}
	if src.Database.Driver != "" {
		dst.DBDriver = normalizeDBDriver(src.Database.Driver)
	}
	if src.Database.SQLitePath != "" {
		dst.SQLitePath = src.Database.SQLitePath
	}
	if src.OSS.Endpoint != "" {
		dst.OSS.Endpoint = src.OSS.Endpoint
	}
	if src.OSS.AccessKey != "" {
		dst.OSS.AccessKey = src.OSS.AccessKey
	}
	if src.OSS.SecretKey != "" {
		dst.OSS.SecretKey = src.OSS.SecretKey
	}
	if src.OSS.Bucket != "" {
		dst.OSS.Bucket = src.OSS.Bucket
	}
	if src.OSS.PublicBaseURL != "" {
		dst.OSS.PublicBaseURL = src.OSS.PublicBaseURL
	}
	if src.OSS.UseSSL != nil {
		dst.OSS.UseSSL = *src.OSS.UseSSL
	}
	if src.UploadCleanup.Enabled != nil {
		dst.UploadCleanup.Enabled = *src.UploadCleanup.Enabled
	}
	if src.UploadCleanup.IntervalSec > 0 {
		dst.UploadCleanup.Interval = time.Duration(src.UploadCleanup.IntervalSec) * time.Second
	}
	if src.UploadCleanup.BatchSize > 0 {
		dst.UploadCleanup.BatchSize = src.UploadCleanup.BatchSize
	}
	if src.Log.Dir != "" {
		dst.Log.Dir = src.Log.Dir
	}
	if src.Log.Level != "" {
		dst.Log.Level = src.Log.Level
	}
	if src.Log.Format != "" {
		dst.Log.Format = src.Log.Format
	}
	if src.Log.ConsoleFormat != "" {
		dst.Log.ConsoleFormat = src.Log.ConsoleFormat
	}
	if src.Log.Console != nil {
		dst.Log.EnableConsole = *src.Log.Console
	}
}

func applyEnvOverrides(cfg *Config, defaultConsoleFormat string) {
	if v := strings.TrimSpace(os.Getenv("ADDR")); v != "" {
		cfg.Addr = v
	}
	if v := strings.TrimSpace(os.Getenv("DB_DSN")); v != "" {
		cfg.DBDSN = v
	}
	if v := strings.TrimSpace(os.Getenv("DB_DRIVER")); v != "" {
		cfg.DBDriver = normalizeDBDriver(v)
	}
	if v := strings.TrimSpace(os.Getenv("SQLITE_PATH")); v != "" {
		cfg.SQLitePath = v
	}
	if v := strings.TrimSpace(os.Getenv("JWT_SECRET")); v != "" {
		cfg.JWTSecret = v
	}
	if v := strings.TrimSpace(os.Getenv("OSS_ENDPOINT")); v != "" {
		cfg.OSS.Endpoint = v
	}
	if v := strings.TrimSpace(os.Getenv("OSS_ACCESS_KEY")); v != "" {
		cfg.OSS.AccessKey = v
	}
	if v := strings.TrimSpace(os.Getenv("OSS_SECRET_KEY")); v != "" {
		cfg.OSS.SecretKey = v
	}
	if v := strings.TrimSpace(os.Getenv("OSS_BUCKET")); v != "" {
		cfg.OSS.Bucket = v
	}
	if v := strings.TrimSpace(os.Getenv("OSS_PUBLIC_BASE_URL")); v != "" {
		cfg.OSS.PublicBaseURL = v
	}
	if v := strings.TrimSpace(os.Getenv("OSS_USE_SSL")); v != "" {
		cfg.OSS.UseSSL = !strings.EqualFold(v, "0") && !strings.EqualFold(v, "false")
	}
	if v := strings.TrimSpace(os.Getenv("UPLOAD_CLEANUP_ENABLED")); v != "" {
		cfg.UploadCleanup.Enabled = !strings.EqualFold(v, "0") && !strings.EqualFold(v, "false")
	}
	if v := strings.TrimSpace(os.Getenv("UPLOAD_CLEANUP_INTERVAL_SEC")); v != "" {
		if sec, err := strconv.Atoi(v); err == nil && sec > 0 {
			cfg.UploadCleanup.Interval = time.Duration(sec) * time.Second
		}
	}
	if v := strings.TrimSpace(os.Getenv("UPLOAD_CLEANUP_BATCH_SIZE")); v != "" {
		if n, err := strconv.Atoi(v); err == nil && n > 0 {
			cfg.UploadCleanup.BatchSize = n
		}
	}
	if strings.TrimSpace(os.Getenv("SKIP_DB")) == "1" {
		cfg.SkipDB = true
	}
	if v := strings.TrimSpace(os.Getenv("LOG_DIR")); v != "" {
		cfg.Log.Dir = v
	}
	if v := strings.TrimSpace(os.Getenv("LOG_LEVEL")); v != "" {
		cfg.Log.Level = v
	}
	if v := strings.TrimSpace(os.Getenv("LOG_FORMAT")); v != "" {
		cfg.Log.Format = v
	}
	if v := strings.TrimSpace(os.Getenv("LOG_CONSOLE_FORMAT")); v != "" {
		cfg.Log.ConsoleFormat = v
	} else if cfg.Log.ConsoleFormat == "" {
		cfg.Log.ConsoleFormat = defaultConsoleFormat
	}
	if v := strings.TrimSpace(os.Getenv("LOG_CONSOLE")); v != "" {
		cfg.Log.EnableConsole = !strings.EqualFold(v, "0") && !strings.EqualFold(v, "false")
	}
}

func normalizeDBDriver(raw string) string {
	switch strings.ToLower(strings.TrimSpace(raw)) {
	case "", DBDriverPostgres:
		return DBDriverPostgres
	case "sqlite3", DBDriverSQLite:
		return DBDriverSQLite
	default:
		return strings.ToLower(strings.TrimSpace(raw))
	}
}
