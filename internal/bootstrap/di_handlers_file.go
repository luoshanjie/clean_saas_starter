package bootstrap

import (
	"service/internal/app/usecase"
	"service/internal/delivery/http/handler"
)

func wireFileHandlers(h *bootstrapHandlers, d handlerDeps) {
	h.fileHandler = &handler.FileHandler{
		UploadSessionCreateUC: &usecase.FileUploadSessionCreateUsecase{
			Storage: d.repos.objectStorage,
			Repo:    d.repos.fileUploadSessionRepo,
			IDGen:   d.idGen,
			Now:     d.now,
		},
		UploadConfirmUC: &usecase.FileUploadConfirmUsecase{
			Repo:     d.repos.fileUploadSessionRepo,
			FileRepo: d.repos.fileRepo,
			Storage:  d.repos.objectStorage,
			Now:      d.now,
		},
		DownloadPresignUC: &usecase.FileDownloadPresignUsecase{
			Storage:  d.repos.objectStorage,
			FileRepo: d.repos.fileRepo,
		},
		CleanupExpiredUC: &usecase.CleanupExpiredUploadSessionsUsecase{
			Storage: d.repos.objectStorage,
			Repo:    d.repos.fileUploadSessionRepo,
			Now:     d.now,
		},
	}
}
