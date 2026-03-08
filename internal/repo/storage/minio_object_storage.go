package storage

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"errors"
	"fmt"
	"net/url"
	"path"
	"strings"
	"time"

	"github.com/minio/minio-go/v7"
	"github.com/minio/minio-go/v7/pkg/credentials"

	domainErr "service/internal/domain/errors"
	"service/internal/domain/port"
)

type MinIOConfig struct {
	Endpoint      string
	AccessKey     string
	SecretKey     string
	Bucket        string
	PublicBaseURL string
	UseSSL        bool
	PresignExpire time.Duration
	Now           func() time.Time
}

type MinIOObjectStorage struct {
	client        *minio.Client
	bucket        string
	publicBaseURL *url.URL
	presignExpire time.Duration
	now           func() time.Time
}

func NewMinIOObjectStorage(cfg MinIOConfig) *MinIOObjectStorage {
	endpointHost, publicBaseURL := normalizeMinIOEndpoint(cfg.Endpoint, cfg.PublicBaseURL, cfg.Bucket, cfg.UseSSL)
	client, err := minio.New(endpointHost, &minio.Options{
		Creds:  credentials.NewStaticV4(strings.TrimSpace(cfg.AccessKey), strings.TrimSpace(cfg.SecretKey), ""),
		Secure: cfg.UseSSL,
	})
	if err != nil {
		return nil
	}
	exp := cfg.PresignExpire
	if exp <= 0 {
		exp = 15 * time.Minute
	}
	nowFn := cfg.Now
	if nowFn == nil {
		nowFn = time.Now
	}
	u, _ := url.Parse(publicBaseURL)
	return &MinIOObjectStorage{
		client:        client,
		bucket:        strings.TrimSpace(cfg.Bucket),
		publicBaseURL: u,
		presignExpire: exp,
		now:           nowFn,
	}
}

func (s *MinIOObjectStorage) PresignUpload(ctx context.Context, in port.PresignUploadInput) (*port.PresignUploadOutput, error) {
	if s == nil || s.client == nil {
		return nil, domainErr.ErrValidation
	}
	bizType := sanitizeSegment(in.BizType)
	fileName := sanitizeFileName(in.FileName)
	if bizType == "" || fileName == "" {
		return nil, domainErr.ErrValidation
	}
	objectKey := buildObjectKey(bizType, fileName, s.now())
	u, err := s.client.PresignedPutObject(ctx, s.bucket, objectKey, s.presignExpire)
	if err != nil {
		return nil, err
	}
	fileURL, err := s.buildFileURL(objectKey)
	if err != nil {
		return nil, err
	}
	return &port.PresignUploadOutput{
		UploadURL: u.String(),
		FileURL:   fileURL,
	}, nil
}

func (s *MinIOObjectStorage) PresignDownload(ctx context.Context, in port.PresignDownloadInput) (*port.PresignDownloadOutput, error) {
	if s == nil || s.client == nil {
		return nil, domainErr.ErrValidation
	}
	objectKey := strings.TrimSpace(in.ObjectKey)
	if objectKey == "" {
		var err error
		objectKey, err = s.objectKeyFromFileURL(strings.TrimSpace(in.FileURL))
		if err != nil {
			return nil, err
		}
	}
	u, err := s.client.PresignedGetObject(ctx, s.bucket, objectKey, s.presignExpire, nil)
	if err != nil {
		return nil, err
	}
	return &port.PresignDownloadOutput{DownloadURL: u.String()}, nil
}

func (s *MinIOObjectStorage) ResolveFile(_ context.Context, in port.ResolveFileInput) (*port.ResolveFileOutput, error) {
	if s == nil || s.client == nil {
		return nil, domainErr.ErrValidation
	}
	objectKey, err := s.objectKeyFromFileURL(strings.TrimSpace(in.FileURL))
	if err != nil {
		return nil, err
	}
	return &port.ResolveFileOutput{
		Bucket:    s.bucket,
		ObjectKey: objectKey,
	}, nil
}

func (s *MinIOObjectStorage) DeleteObject(ctx context.Context, in port.DeleteObjectInput) error {
	if s == nil || s.client == nil {
		return domainErr.ErrValidation
	}
	objectKey := strings.TrimSpace(in.ObjectKey)
	if objectKey == "" {
		return domainErr.ErrValidation
	}
	bucket := strings.TrimSpace(in.Bucket)
	if bucket == "" {
		bucket = s.bucket
	}
	return s.client.RemoveObject(ctx, bucket, objectKey, minio.RemoveObjectOptions{})
}

func (s *MinIOObjectStorage) DeleteByFileURL(ctx context.Context, fileURL string) error {
	if s == nil || s.client == nil {
		return domainErr.ErrValidation
	}
	objectKey, err := s.objectKeyFromFileURL(strings.TrimSpace(fileURL))
	if err != nil {
		return err
	}
	return s.DeleteObject(ctx, port.DeleteObjectInput{
		Bucket:    s.bucket,
		ObjectKey: objectKey,
	})
}

func (s *MinIOObjectStorage) buildFileURL(objectKey string) (string, error) {
	if s.publicBaseURL == nil {
		return "", errors.New("invalid object storage public base url")
	}
	u := *s.publicBaseURL
	basePath := strings.TrimSuffix(u.Path, "/")
	u.Path = basePath + "/" + objectKey
	return u.String(), nil
}

func (s *MinIOObjectStorage) objectKeyFromFileURL(fileURL string) (string, error) {
	if fileURL == "" {
		return "", domainErr.ErrValidation
	}
	u, err := url.Parse(fileURL)
	if err != nil {
		return "", domainErr.ErrValidation
	}
	if s.publicBaseURL == nil {
		return "", domainErr.ErrValidation
	}
	if !strings.EqualFold(u.Host, s.publicBaseURL.Host) {
		return "", domainErr.ErrForbidden
	}
	basePath := strings.TrimSuffix(s.publicBaseURL.Path, "/")
	fullPath := strings.TrimPrefix(u.Path, "/")
	expectedPrefix := strings.TrimPrefix(basePath, "/")
	if expectedPrefix != "" {
		if !strings.HasPrefix(fullPath, expectedPrefix+"/") {
			return "", domainErr.ErrForbidden
		}
		fullPath = strings.TrimPrefix(fullPath, expectedPrefix+"/")
	}
	if strings.TrimSpace(fullPath) == "" {
		return "", domainErr.ErrValidation
	}
	return fullPath, nil
}

func normalizeMinIOEndpoint(endpoint, publicBaseURL, bucket string, useSSL bool) (string, string) {
	raw := strings.TrimSpace(endpoint)
	if raw == "" {
		return "", ""
	}
	scheme := "http"
	if useSSL {
		scheme = "https"
	}
	host := raw
	if parsed, err := url.Parse(raw); err == nil && parsed.Host != "" {
		host = parsed.Host
		if publicBaseURL == "" {
			scheme = parsed.Scheme
		}
	}
	if scheme == "" {
		scheme = "https"
	}
	if publicBaseURL == "" {
		publicBaseURL = fmt.Sprintf("%s://%s/%s", scheme, host, strings.TrimSpace(bucket))
	}
	return host, publicBaseURL
}

func buildObjectKey(bizType, fileName string, now time.Time) string {
	if now.IsZero() {
		now = time.Now()
	}
	dateSeg := now.UTC().Format("20060102")
	return path.Join(bizType, dateSeg, fmt.Sprintf("%d_%s_%s", now.UTC().Unix(), randHex(6), fileName))
}

func sanitizeSegment(s string) string {
	s = strings.ToLower(strings.TrimSpace(s))
	if s == "" {
		return ""
	}
	var b strings.Builder
	b.Grow(len(s))
	lastDash := false
	for _, r := range s {
		ok := (r >= 'a' && r <= 'z') || (r >= '0' && r <= '9')
		if ok {
			b.WriteRune(r)
			lastDash = false
			continue
		}
		if r == '_' || r == '-' {
			if !lastDash {
				b.WriteRune(r)
				lastDash = true
			}
			continue
		}
		if !lastDash {
			b.WriteByte('-')
			lastDash = true
		}
	}
	out := strings.Trim(b.String(), "-_")
	return out
}

func sanitizeFileName(name string) string {
	name = path.Base(strings.TrimSpace(name))
	if name == "." || name == "/" || name == "" {
		return ""
	}
	var b strings.Builder
	b.Grow(len(name))
	for _, r := range name {
		switch {
		case r >= 'a' && r <= 'z':
			b.WriteRune(r)
		case r >= 'A' && r <= 'Z':
			b.WriteRune(r)
		case r >= '0' && r <= '9':
			b.WriteRune(r)
		case r == '.', r == '-', r == '_':
			b.WriteRune(r)
		default:
			b.WriteByte('_')
		}
	}
	out := strings.Trim(b.String(), "._")
	if out == "" {
		return ""
	}
	if len(out) > 120 {
		out = out[len(out)-120:]
	}
	return out
}

func randHex(n int) string {
	if n <= 0 {
		return ""
	}
	buf := make([]byte, n)
	if _, err := rand.Read(buf); err != nil {
		return "rand"
	}
	s := hex.EncodeToString(buf)
	if len(s) > n*2 {
		return s[:n*2]
	}
	return s
}
