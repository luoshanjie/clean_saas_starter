package handler

import (
	"errors"
	"net/http"
	"strings"
	"time"

	"github.com/labstack/echo/v4"

	"service/internal/app/usecase"
	"service/internal/delivery/http/middleware"
	"service/internal/delivery/http/resp"
	domainErr "service/internal/domain/errors"
)

type FileHandler struct {
	UploadSessionCreateUC *usecase.FileUploadSessionCreateUsecase
	UploadConfirmUC       *usecase.FileUploadConfirmUsecase
	DownloadPresignUC     *usecase.FileDownloadPresignUsecase
	DeleteUC              *usecase.FileDeleteUsecase
	CleanupExpiredUC      *usecase.CleanupExpiredUploadSessionsUsecase
}

type fileUploadSessionCreateRequest struct {
	BizType     string `json:"biz_type"`
	FileName    string `json:"file_name"`
	ContentType string `json:"content_type"`
	SizeBytes   int64  `json:"size_bytes"`
}

type fileUploadConfirmRequest struct {
	UploadID     string `json:"upload_id"`
	SizeBytes    int64  `json:"size_bytes"`
	Checksum     string `json:"checksum"`
	ClientSource string `json:"client_source"`
}

type fileDownloadPresignRequest struct {
	FileID  string `json:"file_id"`
	FileURL string `json:"file_url"`
}

type fileUploadCleanupRequest struct {
	Limit int `json:"limit"`
}

type fileDeleteRequest struct {
	FileID string `json:"file_id"`
}

func (h *FileHandler) UploadSessionCreate(c echo.Context) error {
	middleware.SetLogModule(c, "file")
	middleware.SetLogAction(c, "upload.session.create")
	middleware.SetLogBizStep(c, "file.upload_session_create.bind_request")
	var req fileUploadSessionCreateRequest
	if err := c.Bind(&req); err != nil {
		middleware.SetLogFailPoint(c, "bind_request")
		return c.JSON(http.StatusOK, resp.ErrorWithRequestID(middleware.GetRequestID(c), resp.CodeValidation, "bad request"))
	}
	middleware.SetLogBizStep(c, "file.upload_session_create.execute")
	out, err := h.UploadSessionCreateUC.Execute(c.Request().Context(), usecase.FileUploadSessionCreateInput{
		TenantID:    middleware.GetTenantID(c),
		ActorUserID: middleware.GetUserID(c),
		ScopeType:   middleware.GetScopeType(c),
		BizType:     req.BizType,
		FileName:    req.FileName,
		ContentType: req.ContentType,
		SizeBytes:   req.SizeBytes,
	})
	if err != nil {
		middleware.SetLogFailPoint(c, fileFailPoint(err))
		return h.fileError(c, err)
	}
	middleware.SetLogTargetID(c, out.UploadID)
	middleware.SetLogBizStep(c, "file.upload_session_create.success")
	return c.JSON(http.StatusOK, resp.OKWithRequestID(middleware.GetRequestID(c), map[string]any{
		"upload_id":  out.UploadID,
		"upload_url": out.UploadURL,
		"file_url":   out.FileURL,
		"status":     out.Status,
		"expires_at": out.ExpiresAt.UTC().Format(time.RFC3339),
	}))
}

func (h *FileHandler) UploadConfirm(c echo.Context) error {
	middleware.SetLogModule(c, "file")
	middleware.SetLogAction(c, "upload.confirm")
	middleware.SetLogBizStep(c, "file.upload_confirm.bind_request")
	var req fileUploadConfirmRequest
	if err := c.Bind(&req); err != nil {
		middleware.SetLogFailPoint(c, "bind_request")
		return c.JSON(http.StatusOK, resp.ErrorWithRequestID(middleware.GetRequestID(c), resp.CodeValidation, "bad request"))
	}
	middleware.SetLogTargetID(c, strings.TrimSpace(req.UploadID))
	middleware.SetLogBizStep(c, "file.upload_confirm.execute")
	out, err := h.UploadConfirmUC.Execute(c.Request().Context(), usecase.FileUploadConfirmInput{
		UploadID:     req.UploadID,
		TenantID:     middleware.GetTenantID(c),
		ActorUserID:  middleware.GetUserID(c),
		SizeBytes:    req.SizeBytes,
		Checksum:     req.Checksum,
		ClientSource: req.ClientSource,
	})
	if err != nil {
		middleware.SetLogFailPoint(c, fileFailPoint(err))
		return h.fileError(c, err)
	}
	middleware.SetLogTargetID(c, out.FileID)
	middleware.SetLogBizStep(c, "file.upload_confirm.success")
	return c.JSON(http.StatusOK, resp.OKWithRequestID(middleware.GetRequestID(c), map[string]any{
		"status":   out.Status,
		"file_id":  out.FileID,
		"file_url": out.FileURL,
	}))
}

func (h *FileHandler) DownloadPresign(c echo.Context) error {
	var req fileDownloadPresignRequest
	if err := c.Bind(&req); err != nil {
		return c.JSON(http.StatusOK, resp.ErrorWithRequestID(middleware.GetRequestID(c), resp.CodeValidation, "bad request"))
	}
	out, err := h.DownloadPresignUC.Execute(c.Request().Context(), usecase.FileDownloadPresignInput{
		FileID:  req.FileID,
		FileURL: req.FileURL,
	})
	if err != nil {
		return h.fileError(c, err)
	}
	return c.JSON(http.StatusOK, resp.OKWithRequestID(middleware.GetRequestID(c), map[string]any{
		"download_url": out.DownloadURL,
	}))
}

func (h *FileHandler) Delete(c echo.Context) error {
	middleware.SetLogModule(c, "file")
	middleware.SetLogAction(c, "delete")
	var req fileDeleteRequest
	if err := c.Bind(&req); err != nil {
		return c.JSON(http.StatusOK, resp.ErrorWithRequestID(middleware.GetRequestID(c), resp.CodeValidation, "bad request"))
	}
	out, err := h.DeleteUC.Execute(c.Request().Context(), usecase.FileDeleteInput{
		FileID: req.FileID,
	})
	if err != nil {
		return h.fileError(c, err)
	}
	return c.JSON(http.StatusOK, resp.OKWithRequestID(middleware.GetRequestID(c), map[string]any{
		"status": out.Status,
	}))
}

func (h *FileHandler) CleanupExpired(c echo.Context) error {
	var req fileUploadCleanupRequest
	if err := c.Bind(&req); err != nil {
		return c.JSON(http.StatusOK, resp.ErrorWithRequestID(middleware.GetRequestID(c), resp.CodeValidation, "bad request"))
	}
	out, err := h.CleanupExpiredUC.Execute(c.Request().Context(), usecase.CleanupExpiredUploadSessionsInput{
		Limit: req.Limit,
	})
	if err != nil {
		return h.fileError(c, err)
	}
	return c.JSON(http.StatusOK, resp.OKWithRequestID(middleware.GetRequestID(c), map[string]any{
		"scanned": out.Scanned,
		"cleaned": out.Cleaned,
		"failed":  out.Failed,
	}))
}

func (h *FileHandler) fileError(c echo.Context, err error) error {
	switch {
	case errors.Is(err, domainErr.ErrForbidden):
		return c.JSON(http.StatusOK, resp.ErrorWithRequestID(middleware.GetRequestID(c), resp.CodeForbidden, "forbidden"))
	case errors.Is(err, domainErr.ErrValidation):
		return c.JSON(http.StatusOK, resp.ErrorWithRequestID(middleware.GetRequestID(c), resp.CodeValidation, "validation_error"))
	case errors.Is(err, domainErr.ErrNotFound):
		return c.JSON(http.StatusOK, resp.ErrorWithRequestID(middleware.GetRequestID(c), resp.CodeValidation, "not_found"))
	default:
		return c.JSON(http.StatusOK, resp.ErrorWithRequestID(middleware.GetRequestID(c), resp.CodeServerError, err.Error()))
	}
}

func fileFailPoint(err error) string {
	if err == nil {
		return ""
	}
	switch {
	case errors.Is(err, domainErr.ErrForbidden):
		return "forbidden"
	case errors.Is(err, domainErr.ErrNotFound):
		return "upload_session_not_found"
	case strings.Contains(err.Error(), "upload_presign_failed"):
		return "upload_presign"
	case strings.Contains(err.Error(), "upload_session_persist_failed"):
		return "upload_session_persist"
	case strings.Contains(err.Error(), "upload_session_load_failed"):
		return "upload_session_load"
	case strings.Contains(err.Error(), "upload_confirm_persist_failed"):
		return "upload_confirm_persist"
	default:
		if errors.Is(err, domainErr.ErrValidation) {
			return "validation"
		}
		return "internal"
	}
}
