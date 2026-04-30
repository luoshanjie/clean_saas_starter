package bootstrap

import (
	"context"
	"testing"
	"time"
)

func TestInitDB_RejectsSQLite(t *testing.T) {
	runtime, err := InitDB(context.Background(), Config{DBDriver: "sqlite"})
	if err == nil {
		if runtime != nil {
			runtime.Close()
		}
		t.Fatalf("expected sqlite driver to be unsupported")
	}
}

func TestNewBootstrapRepos_DriverAware(t *testing.T) {
	now := func() time.Time { return time.Unix(0, 0) }

	t.Run("nil_database_runtime", func(t *testing.T) {
		_, err := newBootstrapRepos(nil, now, OSSConfig{}, "")
		if err == nil {
			t.Fatalf("expected error for nil database runtime")
		}
	})

	t.Run("sqlite_runtime_unsupported", func(t *testing.T) {
		_, err := newBootstrapRepos(&DBRuntime{Driver: "sqlite"}, now, OSSConfig{}, "")
		if err == nil {
			t.Fatalf("expected sqlite runtime to be unsupported")
		}
	})
}
