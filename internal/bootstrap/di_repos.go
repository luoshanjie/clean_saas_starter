package bootstrap

import (
	"strings"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"

	"service/internal/domain/port"
	cacherepo "service/internal/repo/cache"
	"service/internal/repo/pg"
	storagerepo "service/internal/repo/storage"
)

type bootstrapRepos struct {
	authRepo              *pg.AuthRepoPG
	tenantRepo            *pg.TenantRepoPG
	auditRepo             *pg.AuditRepoPG
	fileUploadSessionRepo *pg.FileUploadSessionRepoPG
	objectStorage         port.ObjectStorage
}

func newBootstrapRepos(pool *pgxpool.Pool, now func() time.Time, ossCfg OSSConfig, _ string) (*bootstrapRepos, error) {
	_, err := cacherepo.NewRistrettoCache(cacherepo.RistrettoConfig{
		NumCounters: 1_000_000,
		MaxCost:     64 << 20,
		BufferItems: 64,
	})
	if err != nil {
		return nil, err
	}
	return &bootstrapRepos{
		authRepo:              &pg.AuthRepoPG{DB: pool},
		tenantRepo:            &pg.TenantRepoPG{DB: pool},
		auditRepo:             &pg.AuditRepoPG{DB: pool},
		fileUploadSessionRepo: &pg.FileUploadSessionRepoPG{DB: pool},
		objectStorage:         newObjectStorage(ossCfg, now),
	}, nil
}

func newObjectStorage(c OSSConfig, now func() time.Time) port.ObjectStorage {
	if strings.TrimSpace(c.Endpoint) == "" ||
		strings.TrimSpace(c.AccessKey) == "" ||
		strings.TrimSpace(c.SecretKey) == "" ||
		strings.TrimSpace(c.Bucket) == "" {
		return &storagerepo.MockObjectStorage{Now: now}
	}
	storage := storagerepo.NewMinIOObjectStorage(storagerepo.MinIOConfig{
		Endpoint:      c.Endpoint,
		AccessKey:     c.AccessKey,
		SecretKey:     c.SecretKey,
		Bucket:        c.Bucket,
		PublicBaseURL: c.PublicBaseURL,
		UseSSL:        c.UseSSL,
		PresignExpire: 15 * time.Minute,
		Now:           now,
	})
	if storage == nil {
		return &storagerepo.MockObjectStorage{Now: now}
	}
	return storage
}
