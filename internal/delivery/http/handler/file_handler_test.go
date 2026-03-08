package handler_test

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/labstack/echo/v4"

	"service/internal/app/usecase"
	"service/internal/delivery/http/handler"
	"service/internal/domain/model"
	"service/internal/domain/port"
)

type testFileStorage struct {
	lastDownloadInput port.PresignDownloadInput
	lastDeleteInput   port.DeleteObjectInput
	deleteErr         error
}

func (m *testFileStorage) PresignUpload(ctx context.Context, in port.PresignUploadInput) (*port.PresignUploadOutput, error) {
	return &port.PresignUploadOutput{}, nil
}

func (m *testFileStorage) PresignDownload(ctx context.Context, in port.PresignDownloadInput) (*port.PresignDownloadOutput, error) {
	m.lastDownloadInput = in
	return &port.PresignDownloadOutput{DownloadURL: "https://download.local/file"}, nil
}

func (m *testFileStorage) ResolveFile(ctx context.Context, in port.ResolveFileInput) (*port.ResolveFileOutput, error) {
	return &port.ResolveFileOutput{}, nil
}

func (m *testFileStorage) DeleteObject(ctx context.Context, in port.DeleteObjectInput) error {
	m.lastDeleteInput = in
	return m.deleteErr
}

func (m *testFileStorage) DeleteByFileURL(ctx context.Context, fileURL string) error { return nil }

type testFileRepo struct {
	file    *model.File
	deleted string
}

func (m *testFileRepo) Create(ctx context.Context, f *model.File) error { return nil }

func (m *testFileRepo) GetByID(ctx context.Context, id string) (*model.File, error) {
	return m.file, nil
}

func (m *testFileRepo) DeleteByID(ctx context.Context, id string) error {
	if m.file == nil {
		return errors.New("not found")
	}
	m.deleted = id
	m.file = nil
	return nil
}

type fileDownloadResp struct {
	Code int `json:"code"`
	Data struct {
		DownloadURL string `json:"download_url"`
	} `json:"data"`
}

type fileDeleteResp struct {
	Code int `json:"code"`
	Data struct {
		Status string `json:"status"`
	} `json:"data"`
}

func TestFileHandler_DownloadPresignByFileID(t *testing.T) {
	e := echo.New()
	storage := &testFileStorage{}
	fileRepo := &testFileRepo{
		file: &model.File{
			ID:        "f1",
			ObjectKey: "biz/20260308/file.png",
		},
	}
	h := &handler.FileHandler{
		DownloadPresignUC: &usecase.FileDownloadPresignUsecase{
			Storage:  storage,
			FileRepo: fileRepo,
		},
	}

	body := []byte(`{"file_id":"f1","file_url":"https://should.not.be.used/example.png"}`)
	req := httptest.NewRequest(http.MethodPost, "/file/download/presign", bytes.NewReader(body))
	req.Header.Set(echo.HeaderContentType, echo.MIMEApplicationJSON)
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)

	if err := h.DownloadPresign(c); err != nil {
		t.Fatalf("handler error: %v", err)
	}

	var resp fileDownloadResp
	if err := json.Unmarshal(rec.Body.Bytes(), &resp); err != nil {
		t.Fatalf("invalid json: %v", err)
	}
	if resp.Code != 0 || resp.Data.DownloadURL == "" {
		t.Fatalf("unexpected response: %+v", resp)
	}
	if storage.lastDownloadInput.ObjectKey != "biz/20260308/file.png" {
		t.Fatalf("expected file_id path to use object key, got %+v", storage.lastDownloadInput)
	}
	if storage.lastDownloadInput.FileURL != "" {
		t.Fatalf("expected file_url to be ignored when file_id is provided, got %+v", storage.lastDownloadInput)
	}
}

func TestFileHandler_DownloadPresignFallbackToFileURL(t *testing.T) {
	e := echo.New()
	storage := &testFileStorage{}
	h := &handler.FileHandler{
		DownloadPresignUC: &usecase.FileDownloadPresignUsecase{
			Storage: storage,
		},
	}

	body := []byte(`{"file_url":"https://storage.local/bucket/path/to/file.png"}`)
	req := httptest.NewRequest(http.MethodPost, "/file/download/presign", bytes.NewReader(body))
	req.Header.Set(echo.HeaderContentType, echo.MIMEApplicationJSON)
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)

	if err := h.DownloadPresign(c); err != nil {
		t.Fatalf("handler error: %v", err)
	}

	var resp fileDownloadResp
	if err := json.Unmarshal(rec.Body.Bytes(), &resp); err != nil {
		t.Fatalf("invalid json: %v", err)
	}
	if resp.Code != 0 || resp.Data.DownloadURL == "" {
		t.Fatalf("unexpected response: %+v", resp)
	}
	if storage.lastDownloadInput.FileURL != "https://storage.local/bucket/path/to/file.png" {
		t.Fatalf("expected file_url fallback, got %+v", storage.lastDownloadInput)
	}
}

func TestFileHandler_Delete(t *testing.T) {
	e := echo.New()
	storage := &testFileStorage{}
	fileRepo := &testFileRepo{
		file: &model.File{
			ID:        "f1",
			Bucket:    "bucket-1",
			ObjectKey: "biz/20260308/file.png",
		},
	}
	h := &handler.FileHandler{
		DeleteUC: &usecase.FileDeleteUsecase{
			Storage:  storage,
			FileRepo: fileRepo,
		},
	}

	body := []byte(`{"file_id":"f1"}`)
	req := httptest.NewRequest(http.MethodPost, "/file/delete", bytes.NewReader(body))
	req.Header.Set(echo.HeaderContentType, echo.MIMEApplicationJSON)
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)

	if err := h.Delete(c); err != nil {
		t.Fatalf("handler error: %v", err)
	}

	var resp fileDeleteResp
	if err := json.Unmarshal(rec.Body.Bytes(), &resp); err != nil {
		t.Fatalf("invalid json: %v", err)
	}
	if resp.Code != 0 || resp.Data.Status != "ok" {
		t.Fatalf("unexpected response: %+v", resp)
	}
	if storage.lastDeleteInput.ObjectKey != "biz/20260308/file.png" {
		t.Fatalf("unexpected delete object input: %+v", storage.lastDeleteInput)
	}
	if fileRepo.deleted != "f1" {
		t.Fatalf("expected file record delete, got %q", fileRepo.deleted)
	}
}
