package usecase

import (
	"context"
	"errors"
	"strings"
	"time"

	domainErr "service/internal/domain/errors"
	"service/internal/domain/model"
	"service/internal/domain/port"
)

type AuditWriteInput struct {
	RequestID           string
	OperatorUserID      string
	OperatorRole        string
	OperatorTenantID    string
	OperatorUsername    string
	OperatorDisplayName string
	TargetType          string
	TargetID            string
	TargetName          string
	Action              string
	Module              string
	Result              string
	ErrorCode           string
	BeforeJSON          map[string]any
	AfterJSON           map[string]any
	ChangedFields       []string
	IP                  string
	UserAgent           string
}

type AuditWriteUsecase struct {
	Repo  port.AuditRepo
	IDGen func() string
	Now   func() time.Time
}

func (u *AuditWriteUsecase) LogSafe(ctx context.Context, in AuditWriteInput) {
	if u == nil || u.Repo == nil || u.IDGen == nil || strings.TrimSpace(in.TargetType) == "" || strings.TrimSpace(in.Action) == "" || strings.TrimSpace(in.Module) == "" {
		return
	}
	result := in.Result
	if result == "" {
		result = "success"
	}
	now := time.Now()
	if u.Now != nil {
		now = u.Now()
	}
	id, err := canonicalUUIDText(u.IDGen())
	if err != nil {
		return
	}
	_ = u.Repo.Create(ctx, &model.AuditLog{
		ID:                  id,
		RequestID:           in.RequestID,
		OperatorUserID:      in.OperatorUserID,
		OperatorRole:        in.OperatorRole,
		OperatorTenantID:    in.OperatorTenantID,
		OperatorUsername:    in.OperatorUsername,
		OperatorDisplayName: in.OperatorDisplayName,
		TargetType:          in.TargetType,
		TargetID:            in.TargetID,
		TargetName:          in.TargetName,
		Action:              in.Action,
		Module:              in.Module,
		Result:              result,
		ErrorCode:           in.ErrorCode,
		BeforeJSON:          sanitizeAuditMap(in.BeforeJSON),
		AfterJSON:           sanitizeAuditMap(in.AfterJSON),
		ChangedFields:       in.ChangedFields,
		IP:                  in.IP,
		UserAgent:           in.UserAgent,
		CreatedAt:           now,
	})
}

type ListAuditLogsInput struct {
	Role           string
	UserID         string
	TenantID       string
	Keyword        string
	Module         string
	Action         string
	Result         string
	OperatorUserID string
	TargetType     string
	TargetID       string
	RequestID      string
	FilterTenantID string
	DateFrom       string
	DateTo         string
	NeedTotal      bool
	Page           int
	PageSize       int
}

type ListAuditLogsOutput struct {
	Items []*model.AuditLog
	Total int
}

type ListAuditLogsUsecase struct {
	Repo port.AuditRepo
}

func (u *ListAuditLogsUsecase) Execute(ctx context.Context, in ListAuditLogsInput) (*ListAuditLogsOutput, error) {
	filter := port.AuditFilter{
		Keyword:        strings.TrimSpace(in.Keyword),
		Module:         strings.TrimSpace(in.Module),
		Action:         strings.TrimSpace(in.Action),
		Result:         strings.TrimSpace(in.Result),
		OperatorUserID: strings.TrimSpace(in.OperatorUserID),
		TargetType:     strings.TrimSpace(in.TargetType),
		TargetID:       strings.TrimSpace(in.TargetID),
		RequestID:      strings.TrimSpace(in.RequestID),
		TenantID:       strings.TrimSpace(in.FilterTenantID),
		DateFrom:       strings.TrimSpace(in.DateFrom),
		DateTo:         strings.TrimSpace(in.DateTo),
		NeedTotal:      in.NeedTotal,
		Page:           in.Page,
		PageSize:       in.PageSize,
	}
	switch in.Role {
	case "platform_op":
		// full access
	case "tenant_admin":
		if strings.TrimSpace(in.TenantID) == "" {
			return nil, domainErr.ErrForbidden
		}
		filter.TenantID = in.TenantID
	case "tenant_member":
		if strings.TrimSpace(in.TenantID) == "" {
			return nil, domainErr.ErrForbidden
		}
		filter.TenantID = in.TenantID
		filter.OperatorUserID = in.UserID
	default:
		return nil, domainErr.ErrForbidden
	}
	items, total, err := u.Repo.ListPage(ctx, filter)
	if err != nil {
		return nil, err
	}
	return &ListAuditLogsOutput{Items: items, Total: total}, nil
}

type GetAuditDetailInput struct {
	Role     string
	UserID   string
	TenantID string
	AuditID  string
}

type GetAuditDetailOutput struct {
	Log *model.AuditLog
}

type GetAuditDetailUsecase struct {
	Repo port.AuditRepo
}

func (u *GetAuditDetailUsecase) Execute(ctx context.Context, in GetAuditDetailInput) (*GetAuditDetailOutput, error) {
	if strings.TrimSpace(in.AuditID) == "" {
		return nil, domainErr.ErrValidation
	}
	log, err := u.Repo.GetByID(ctx, strings.TrimSpace(in.AuditID))
	if err != nil {
		if errors.Is(err, domainErr.ErrNotFound) {
			return nil, domainErr.ErrNotFound
		}
		return nil, err
	}
	switch in.Role {
	case "platform_op":
		// pass
	case "tenant_admin":
		if strings.TrimSpace(in.TenantID) == "" {
			return nil, domainErr.ErrForbidden
		}
		if log.OperatorTenantID != in.TenantID {
			return nil, domainErr.ErrForbidden
		}
	case "tenant_member":
		if strings.TrimSpace(in.TenantID) == "" {
			return nil, domainErr.ErrForbidden
		}
		if log.OperatorTenantID != in.TenantID || log.OperatorUserID != in.UserID {
			return nil, domainErr.ErrForbidden
		}
	default:
		return nil, domainErr.ErrForbidden
	}
	return &GetAuditDetailOutput{Log: log}, nil
}
