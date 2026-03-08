package port

import "context"

type PresignUploadInput struct {
	BizType     string
	FileName    string
	ContentType string
}

type PresignUploadOutput struct {
	UploadURL string
	FileURL   string
}

type PresignDownloadInput struct {
	FileURL string
}

type PresignDownloadOutput struct {
	DownloadURL string
}

type ObjectStorage interface {
	PresignUpload(ctx context.Context, in PresignUploadInput) (*PresignUploadOutput, error)
	PresignDownload(ctx context.Context, in PresignDownloadInput) (*PresignDownloadOutput, error)
	DeleteByFileURL(ctx context.Context, fileURL string) error
}
