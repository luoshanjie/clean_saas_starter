package pg

import (
	"context"
	"encoding/json"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"service/internal/domain/model"
	"service/internal/domain/port"
	"service/internal/repo/pg/sqlcpg"
)

type AuditRepoPG struct {
	DB *pgxpool.Pool
}

func (r *AuditRepoPG) Create(ctx context.Context, log *model.AuditLog) error {
	beforeVal, _ := marshalOptionalJSONMap(log.BeforeJSON)
	afterVal, _ := marshalOptionalJSONMap(log.AfterJSON)
	changedBytes, _ := marshalStringList(log.ChangedFields)
	if len(changedBytes) == 0 || string(changedBytes) == "null" {
		changedBytes = []byte("[]")
	}

	return withRLS(ctx, r.DB, func(tx pgx.Tx) error {
		q := sqlcpg.New(tx)
		id, err := parseUUIDText(log.ID)
		if err != nil {
			return err
		}
		return q.AuditCreate(ctx, sqlcpg.AuditCreateParams{
			ID:                  id,
			RequestID:           log.RequestID,
			OperatorUserID:      log.OperatorUserID,
			OperatorRole:        log.OperatorRole,
			OperatorTenantID:    log.OperatorTenantID,
			OperatorUsername:    log.OperatorUsername,
			OperatorDisplayName: log.OperatorDisplayName,
			TargetType:          log.TargetType,
			TargetID:            log.TargetID,
			TargetName:          log.TargetName,
			Action:              log.Action,
			Module:              log.Module,
			Result:              log.Result,
			ErrorCode:           log.ErrorCode,
			BeforeJson:          beforeVal,
			AfterJson:           afterVal,
			ChangedFields:       changedBytes,
			Ip:                  log.IP,
			UserAgent:           log.UserAgent,
			CreatedAt:           pgTimestamptz(log.CreatedAt),
		})
	})
}

func (r *AuditRepoPG) ListPage(ctx context.Context, filter port.AuditFilter) ([]*model.AuditLog, int, error) {
	out := make([]*model.AuditLog, 0)
	total := 0
	page, pageSize := normalizePage(filter.Page, filter.PageSize)
	offset := (page - 1) * pageSize
	err := withRLS(ctx, r.DB, func(tx pgx.Tx) error {
		q := sqlcpg.New(tx)
		params := sqlcpg.AuditListPageParams{
			Module:         filter.Module,
			Action:         filter.Action,
			Result:         filter.Result,
			OperatorUserID: filter.OperatorUserID,
			TargetType:     filter.TargetType,
			TargetID:       filter.TargetID,
			RequestID:      filter.RequestID,
			TenantID:       filter.TenantID,
			DateFrom:       filter.DateFrom,
			DateTo:         filter.DateTo,
			Keyword:        filter.Keyword,
			OffsetRows:     int32(offset),
			LimitRows:      int32(pageSize),
		}
		if filter.NeedTotal {
			c, err := q.AuditCount(ctx, sqlcpg.AuditCountParams{
				Module:         filter.Module,
				Action:         filter.Action,
				Result:         filter.Result,
				OperatorUserID: filter.OperatorUserID,
				TargetType:     filter.TargetType,
				TargetID:       filter.TargetID,
				RequestID:      filter.RequestID,
				TenantID:       filter.TenantID,
				DateFrom:       filter.DateFrom,
				DateTo:         filter.DateTo,
				Keyword:        filter.Keyword,
			})
			if err != nil {
				return err
			}
			total = int(c)
		} else {
			total = -1
		}
		rows, err := q.AuditListPage(ctx, params)
		if err != nil {
			return err
		}
		for _, row := range rows {
			log, err := auditListRowToModel(row)
			if err != nil {
				return err
			}
			out = append(out, log)
		}
		return nil
	})
	return out, total, err
}

func (r *AuditRepoPG) GetByID(ctx context.Context, id string) (*model.AuditLog, error) {
	var out *model.AuditLog
	err := withRLS(ctx, r.DB, func(tx pgx.Tx) error {
		q := sqlcpg.New(tx)
		uid, err := parseUUIDText(id)
		if err != nil {
			return err
		}
		row, err := q.AuditGetByID(ctx, uid)
		if err != nil {
			return mapRowNotFound(err)
		}
		log, err := auditGetRowToModel(row)
		if err != nil {
			return err
		}
		out = log
		return nil
	})
	return out, err
}

func marshalOptionalJSONMap(v map[string]any) ([]byte, error) {
	if v == nil {
		return nil, nil
	}
	b, err := json.Marshal(v)
	if err != nil {
		return nil, err
	}
	if string(b) == "null" {
		return nil, nil
	}
	return b, nil
}

func unmarshalOptionalJSONMap(data []byte) (map[string]any, error) {
	if len(data) == 0 || string(data) == "null" {
		return nil, nil
	}
	var out map[string]any
	if err := json.Unmarshal(data, &out); err != nil {
		return nil, err
	}
	return out, nil
}

func unmarshalStringSlice(data []byte) ([]string, error) {
	if len(data) == 0 || string(data) == "null" {
		return []string{}, nil
	}
	var out []string
	if err := json.Unmarshal(data, &out); err != nil {
		return nil, err
	}
	if out == nil {
		out = []string{}
	}
	return out, nil
}

func auditFromRaw(
	id, requestID string,
	operatorUserID any,
	operatorRole string,
	operatorTenantID any,
	operatorUsername, operatorDisplayName, targetType, targetID string,
	targetName any,
	action, module, result, errorCode string,
	beforeJSON, afterJSON, changedFields []byte,
	ip, userAgent string,
	createdAt time.Time,
) (*model.AuditLog, error) {
	beforeMap, err := unmarshalOptionalJSONMap(beforeJSON)
	if err != nil {
		return nil, err
	}
	afterMap, err := unmarshalOptionalJSONMap(afterJSON)
	if err != nil {
		return nil, err
	}
	changed, err := unmarshalStringSlice(changedFields)
	if err != nil {
		return nil, err
	}
	return &model.AuditLog{
		ID:                  id,
		RequestID:           requestID,
		OperatorUserID:      toString(operatorUserID),
		OperatorRole:        operatorRole,
		OperatorTenantID:    toString(operatorTenantID),
		OperatorUsername:    operatorUsername,
		OperatorDisplayName: operatorDisplayName,
		TargetType:          targetType,
		TargetID:            targetID,
		TargetName:          toString(targetName),
		Action:              action,
		Module:              module,
		Result:              result,
		ErrorCode:           errorCode,
		BeforeJSON:          beforeMap,
		AfterJSON:           afterMap,
		ChangedFields:       changed,
		IP:                  ip,
		UserAgent:           userAgent,
		CreatedAt:           createdAt,
	}, nil
}

func auditListRowToModel(row sqlcpg.AuditListPageRow) (*model.AuditLog, error) {
	return auditFromRaw(
		row.ID, row.RequestID,
		row.OperatorUserID, row.OperatorRole, row.OperatorTenantID,
		row.OperatorUsername, row.OperatorDisplayName, row.TargetType, row.TargetID,
		row.TargetName,
		row.Action, row.Module, row.Result, row.ErrorCode,
		row.BeforeJson, row.AfterJson, row.ChangedFields,
		row.Ip, row.UserAgent, row.CreatedAt.Time,
	)
}

func auditGetRowToModel(row sqlcpg.AuditGetByIDRow) (*model.AuditLog, error) {
	return auditFromRaw(
		row.ID, row.RequestID,
		row.OperatorUserID, row.OperatorRole, row.OperatorTenantID,
		row.OperatorUsername, row.OperatorDisplayName, row.TargetType, row.TargetID,
		row.TargetName,
		row.Action, row.Module, row.Result, row.ErrorCode,
		row.BeforeJson, row.AfterJson, row.ChangedFields,
		row.Ip, row.UserAgent, row.CreatedAt.Time,
	)
}
