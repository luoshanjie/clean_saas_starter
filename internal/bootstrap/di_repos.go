package bootstrap

import (
	"context"
	"errors"
	"time"

	"service/internal/domain/port"
	cacherepo "service/internal/repo/cache"
	"service/internal/repo/pg"
	storagerepo "service/internal/repo/storage"
)

type bootstrapAuthRepo interface {
	port.AuthRepo
	port.AuthChallengeRepo
	port.AuthPhoneRepo
	port.PhoneChangeChallengeRepo
	UpdateDisplayNameByUserID(ctx context.Context, userID, name string) error
}

type bootstrapRepos struct {
	authRepo              bootstrapAuthRepo
	tenantRepo            port.TenantRepo
	auditRepo             port.AuditRepo
	fileRepo              port.FileRepo
	fileUploadSessionRepo port.FileUploadSessionRepo
	objectStorage         port.ObjectStorage
}

func newBootstrapRepos(db *DBRuntime, now func() time.Time, ossCfg OSSConfig, _ string) (*bootstrapRepos, error) {
	if db == nil {
		return nil, errors.New("nil database runtime")
	}
	_, err := cacherepo.NewRistrettoCache(cacherepo.RistrettoConfig{
		NumCounters: 1_000_000,
		MaxCost:     64 << 20,
		BufferItems: 64,
	})
	if err != nil {
		return nil, err
	}
	if db.Driver != DBDriverPostgres {
		return nil, errors.New("unsupported database runtime: " + db.Driver)
	}
	if db.Postgres == nil {
		return nil, errors.New("nil postgres pool")
	}
	return &bootstrapRepos{
		authRepo:              &pg.AuthRepoPG{DB: db.Postgres},
		tenantRepo:            &pg.TenantRepoPG{DB: db.Postgres},
		auditRepo:             &pg.AuditRepoPG{DB: db.Postgres},
		fileRepo:              &pg.FileRepoPG{DB: db.Postgres},
		fileUploadSessionRepo: &pg.FileUploadSessionRepoPG{DB: db.Postgres},
		objectStorage:         newObjectStorage(ossCfg, now),
	}, nil
}

func newObjectStorage(c OSSConfig, now func() time.Time) port.ObjectStorage {
	if !c.Enabled() {
		return nil
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
		return nil
	}
	return storage
}
