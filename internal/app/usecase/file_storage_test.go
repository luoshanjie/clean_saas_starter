package usecase

import (
	"context"
	"testing"
	"time"

	domainErr "service/internal/domain/errors"
	"service/internal/domain/model"
	"service/internal/domain/port"
)

type mockFileStorage struct {
	resolveOut         *port.ResolveFileOutput
	presignDownloadOut *port.PresignDownloadOutput
	lastDownloadInput  port.PresignDownloadInput
}

func (m *mockFileStorage) PresignUpload(ctx context.Context, in port.PresignUploadInput) (*port.PresignUploadOutput, error) {
	return &port.PresignUploadOutput{}, nil
}

func (m *mockFileStorage) PresignDownload(ctx context.Context, in port.PresignDownloadInput) (*port.PresignDownloadOutput, error) {
	m.lastDownloadInput = in
	if m.presignDownloadOut != nil {
		return m.presignDownloadOut, nil
	}
	return &port.PresignDownloadOutput{DownloadURL: "https://download.local"}, nil
}

func (m *mockFileStorage) ResolveFile(ctx context.Context, in port.ResolveFileInput) (*port.ResolveFileOutput, error) {
	if m.resolveOut != nil {
		return m.resolveOut, nil
	}
	return &port.ResolveFileOutput{Bucket: "b1", ObjectKey: "obj-1"}, nil
}

func (m *mockFileStorage) DeleteByFileURL(ctx context.Context, fileURL string) error { return nil }

type mockFileUploadSessionRepo struct {
	session *model.FileUploadSession
}

func (m *mockFileUploadSessionRepo) Create(ctx context.Context, s *model.FileUploadSession) error {
	return nil
}
func (m *mockFileUploadSessionRepo) GetByID(ctx context.Context, id string) (*model.FileUploadSession, error) {
	return m.session, nil
}
func (m *mockFileUploadSessionRepo) Confirm(ctx context.Context, id string, confirmedAt time.Time) (*model.FileUploadSession, error) {
	if m.session != nil {
		m.session.Status = "confirmed"
		m.session.ConfirmedAt = &confirmedAt
	}
	return m.session, nil
}
func (m *mockFileUploadSessionRepo) ListExpiredPending(ctx context.Context, now time.Time, limit int) ([]*model.FileUploadSession, error) {
	return nil, nil
}
func (m *mockFileUploadSessionRepo) MarkCleaned(ctx context.Context, id string, cleanedAt time.Time) error {
	return nil
}
func (m *mockFileUploadSessionRepo) SetLastError(ctx context.Context, id, lastError string, updatedAt time.Time) error {
	return nil
}

type mockFileRepo struct {
	file     *model.File
	createIn *model.File
	getErr   error
}

func (m *mockFileRepo) Create(ctx context.Context, f *model.File) error {
	cp := *f
	m.createIn = &cp
	m.file = &cp
	return nil
}

func (m *mockFileRepo) GetByID(ctx context.Context, id string) (*model.File, error) {
	if m.getErr != nil {
		return nil, m.getErr
	}
	if m.file == nil {
		return nil, domainErr.ErrNotFound
	}
	return m.file, nil
}

func TestFileUploadConfirmUsecase_CreatesFileAsset(t *testing.T) {
	now := time.Date(2026, 3, 8, 10, 0, 0, 0, time.UTC)
	repo := &mockFileUploadSessionRepo{
		session: &model.FileUploadSession{
			ID:          "f1",
			TenantID:    "t1",
			UploadedBy:  "u1",
			FileURL:     "https://storage.local/bucket/path/to/file.png",
			ContentType: "image/png",
			SizeBytes:   123,
			Status:      "pending_confirm",
			ExpiresAt:   now.Add(time.Hour),
			CreatedAt:   now,
		},
	}
	fileRepo := &mockFileRepo{}
	storage := &mockFileStorage{
		resolveOut: &port.ResolveFileOutput{
			Bucket:    "bucket",
			ObjectKey: "path/to/file.png",
		},
	}
	uc := &FileUploadConfirmUsecase{
		Repo:     repo,
		FileRepo: fileRepo,
		Storage:  storage,
		Now:      func() time.Time { return now },
	}

	out, err := uc.Execute(context.Background(), FileUploadConfirmInput{
		UploadID:    "f1",
		ActorUserID: "u1",
	})
	if err != nil {
		t.Fatalf("Execute error: %v", err)
	}
	if out.FileID != "f1" || out.FileURL == "" {
		t.Fatalf("unexpected output: %+v", out)
	}
	if fileRepo.createIn == nil {
		t.Fatalf("expected file asset to be created")
	}
	if fileRepo.createIn.ObjectKey != "path/to/file.png" || fileRepo.createIn.Bucket != "bucket" {
		t.Fatalf("unexpected file asset: %+v", fileRepo.createIn)
	}
}

func TestFileDownloadPresignUsecase_LoadsByFileID(t *testing.T) {
	fileRepo := &mockFileRepo{
		file: &model.File{
			ID:        "f1",
			ObjectKey: "biz/20260308/file.png",
		},
	}
	storage := &mockFileStorage{
		presignDownloadOut: &port.PresignDownloadOutput{DownloadURL: "https://download.local/file"},
	}
	uc := &FileDownloadPresignUsecase{
		Storage:  storage,
		FileRepo: fileRepo,
	}

	out, err := uc.Execute(context.Background(), FileDownloadPresignInput{FileID: "f1"})
	if err != nil {
		t.Fatalf("Execute error: %v", err)
	}
	if out.DownloadURL == "" {
		t.Fatalf("expected download url")
	}
	if storage.lastDownloadInput.ObjectKey != "biz/20260308/file.png" {
		t.Fatalf("expected object key download, got %+v", storage.lastDownloadInput)
	}
}
