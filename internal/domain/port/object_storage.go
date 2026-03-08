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
	FileURL   string
	ObjectKey string
}

type PresignDownloadOutput struct {
	DownloadURL string
}

type ResolveFileInput struct {
	FileURL string
}

type ResolveFileOutput struct {
	Bucket    string
	ObjectKey string
}

type DeleteObjectInput struct {
	Bucket    string
	ObjectKey string
}

type ObjectStorage interface {
	PresignUpload(ctx context.Context, in PresignUploadInput) (*PresignUploadOutput, error)
	PresignDownload(ctx context.Context, in PresignDownloadInput) (*PresignDownloadOutput, error)
	ResolveFile(ctx context.Context, in ResolveFileInput) (*ResolveFileOutput, error)
	DeleteObject(ctx context.Context, in DeleteObjectInput) error
	DeleteByFileURL(ctx context.Context, fileURL string) error
}
