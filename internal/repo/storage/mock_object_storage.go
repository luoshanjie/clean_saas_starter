package storage

import (
	"context"
	"strings"
	"time"

	domainErr "service/internal/domain/errors"
	"service/internal/domain/port"
)

type MockObjectStorage struct {
	Now func() time.Time
}

func (m *MockObjectStorage) PresignUpload(_ context.Context, in port.PresignUploadInput) (*port.PresignUploadOutput, error) {
	name := strings.TrimSpace(in.FileName)
	if name == "" || strings.TrimSpace(in.BizType) == "" {
		return nil, domainErr.ErrValidation
	}
	now := time.Now()
	if m != nil && m.Now != nil {
		now = m.Now()
	}
	ts := now.Unix()
	return &port.PresignUploadOutput{
		UploadURL: "https://mock-storage.local/upload/" + strings.TrimSpace(in.BizType) + "/" + name + "?ts=" + int64ToStringStorage(ts),
		FileURL:   "https://mock-storage.local/file/" + strings.TrimSpace(in.BizType) + "/" + name + "?ts=" + int64ToStringStorage(ts),
	}, nil
}

func (m *MockObjectStorage) PresignDownload(_ context.Context, in port.PresignDownloadInput) (*port.PresignDownloadOutput, error) {
	if objectKey := strings.TrimSpace(in.ObjectKey); objectKey != "" {
		return &port.PresignDownloadOutput{DownloadURL: "https://mock-storage.local/download/" + objectKey}, nil
	}
	u := strings.TrimSpace(in.FileURL)
	if u == "" {
		return nil, domainErr.ErrValidation
	}
	return &port.PresignDownloadOutput{DownloadURL: u}, nil
}

func (m *MockObjectStorage) ResolveFile(_ context.Context, in port.ResolveFileInput) (*port.ResolveFileOutput, error) {
	fileURL := strings.TrimSpace(in.FileURL)
	if fileURL == "" {
		return nil, domainErr.ErrValidation
	}
	parts := strings.Split(fileURL, "/file/")
	if len(parts) != 2 {
		return nil, domainErr.ErrValidation
	}
	objectKey := parts[1]
	if idx := strings.IndexByte(objectKey, '?'); idx >= 0 {
		objectKey = objectKey[:idx]
	}
	objectKey = strings.TrimSpace(objectKey)
	if objectKey == "" {
		return nil, domainErr.ErrValidation
	}
	return &port.ResolveFileOutput{
		Bucket:    "mock",
		ObjectKey: objectKey,
	}, nil
}

func (m *MockObjectStorage) DeleteObject(_ context.Context, in port.DeleteObjectInput) error {
	if strings.TrimSpace(in.ObjectKey) == "" {
		return domainErr.ErrValidation
	}
	return nil
}

func (m *MockObjectStorage) DeleteByFileURL(_ context.Context, fileURL string) error {
	if strings.TrimSpace(fileURL) == "" {
		return domainErr.ErrValidation
	}
	return nil
}

func int64ToStringStorage(v int64) string {
	if v == 0 {
		return "0"
	}
	neg := v < 0
	if neg {
		v = -v
	}
	buf := [20]byte{}
	i := len(buf)
	for v > 0 {
		i--
		buf[i] = byte('0' + v%10)
		v /= 10
	}
	if neg {
		i--
		buf[i] = '-'
	}
	return string(buf[i:])
}
