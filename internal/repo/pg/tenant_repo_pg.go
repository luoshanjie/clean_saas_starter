package pg

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/jackc/pgx/v5/pgxpool"

	domainErr "service/internal/domain/errors"
	"service/internal/domain/model"
	"service/internal/domain/port"
	"service/internal/repo/pg/sqlcpg"
)

type TenantRepoPG struct {
	DB *pgxpool.Pool
}

func (r *TenantRepoPG) CreateWithAdmin(ctx context.Context, in *model.TenantCreateInput) (*model.TenantCreateOutput, error) {
	if in == nil {
		return nil, domainErr.ErrValidation
	}
	out := &model.TenantCreateOutput{}
	err := withRLS(ctx, r.DB, func(tx pgx.Tx) error {
		q := sqlcpg.New(tx)

		tenantID, err := parseUUIDText(in.TenantID)
		if err != nil {
			return err
		}
		adminUserID, err := parseUUIDText(in.TenantAdminUserID)
		if err != nil {
			return err
		}

		err = q.CreateTenant(ctx, sqlcpg.CreateTenantParams{
			ID:           tenantID,
			DisplayName:  in.DisplayName,
			Province:     in.Province,
			City:         in.City,
			District:     in.District,
			Address:      in.Address,
			ContactName:  in.ContactName,
			ContactPhone: in.ContactPhone,
			Remark:       in.Remark,
			Status:       in.Status,
			CreatedAt:    pgTimestamptz(in.CreatedAt),
			UpdatedAt:    pgTimestamptz(in.UpdatedAt),
		})
		if err != nil {
			return mapTenantPGError(err)
		}

		err = q.CreateTenantAdminUser(ctx, sqlcpg.CreateTenantAdminUserParams{
			UserID:    adminUserID,
			TenantID:  tenantID,
			Name:      in.TenantAdminName,
			Phone:     in.TenantAdminPhone,
			Status:    in.Status,
			CreatedAt: pgTimestamptz(in.TenantAdminCreatedAt),
		})
		if err != nil {
			return mapTenantPGError(err)
		}

		err = q.CreateTenantAdminCredential(ctx, sqlcpg.CreateTenantAdminCredentialParams{
			UserID:       adminUserID,
			Account:      in.TenantAdminAccount,
			PasswordHash: in.TenantAdminPassword,
			CreatedAt:    pgTimestamptz(in.TenantAdminCreatedAt),
		})
		if err != nil {
			return mapTenantPGError(err)
		}

		out.TenantID = in.TenantID
		out.TenantAdminUserID = in.TenantAdminUserID
		out.TenantAdminAccount = in.TenantAdminAccount
		out.TenantAdminName = in.TenantAdminName
		out.Status = in.Status
		out.CreatedAt = in.CreatedAt
		return nil
	})
	if err != nil {
		return nil, err
	}
	return out, nil
}

func (r *TenantRepoPG) ListPage(ctx context.Context, filter port.TenantFilter) ([]*model.TenantListItem, int, error) {
	out := make([]*model.TenantListItem, 0)
	total := 0
	page, pageSize := normalizePage(filter.Page, filter.PageSize)
	offset := (page - 1) * pageSize
	err := withRLS(ctx, r.DB, func(tx pgx.Tx) error {
		q := sqlcpg.New(tx)

		if filter.NeedTotal {
			count, err := q.CountTenants(ctx, sqlcpg.CountTenantsParams{
				Keyword:  filter.Keyword,
				Province: filter.Province,
				City:     filter.City,
				District: filter.District,
				Status:   filter.Status,
			})
			if err != nil {
				return err
			}
			total = int(count)
		} else {
			total = -1
		}

		rows, err := q.ListTenantsPage(ctx, sqlcpg.ListTenantsPageParams{
			Keyword:    filter.Keyword,
			Province:   filter.Province,
			City:       filter.City,
			District:   filter.District,
			Status:     filter.Status,
			OffsetRows: int32(offset),
			LimitRows:  int32(pageSize),
		})
		if err != nil {
			return err
		}

		for _, row := range rows {
			out = append(out, &model.TenantListItem{
				Tenant: &model.Tenant{
					ID:           row.TenantID,
					DisplayName:  row.DisplayName,
					Province:     row.Province,
					City:         row.City,
					District:     row.District,
					Address:      row.Address,
					ContactName:  row.ContactName,
					ContactPhone: row.ContactPhone,
					Remark:       row.Remark,
					Status:       row.Status,
					CreatedAt:    row.CreatedAt.Time,
					UpdatedAt:    row.UpdatedAt.Time,
				},
				TenantAdminUserID:  toString(row.TenantAdminUserID),
				TenantAdminAccount: row.TenantAdminAccount,
				TenantAdminName:    row.TenantAdminName,
				TenantAdminPhone:   row.TenantAdminPhone,
			})
		}
		return nil
	})
	return out, total, err
}

func (r *TenantRepoPG) GetByID(ctx context.Context, tenantID string) (*model.Tenant, error) {
	var out *model.Tenant
	err := withRLS(ctx, r.DB, func(tx pgx.Tx) error {
		q := sqlcpg.New(tx)

		tenantUUID, err := parseUUIDText(tenantID)
		if err != nil {
			return err
		}

		row, err := q.GetTenantByID(ctx, tenantUUID)
		if err != nil {
			return mapRowNotFound(err)
		}
		out = &model.Tenant{
			ID:           row.TenantID,
			DisplayName:  row.DisplayName,
			Province:     row.Province,
			City:         row.City,
			District:     row.District,
			Address:      row.Address,
			ContactName:  row.ContactName,
			ContactPhone: row.ContactPhone,
			Remark:       row.Remark,
			Status:       row.Status,
			CreatedAt:    row.CreatedAt.Time,
			UpdatedAt:    row.UpdatedAt.Time,
		}
		return nil
	})
	return out, err
}

func (r *TenantRepoPG) Update(ctx context.Context, tenant *model.Tenant) (bool, error) {
	updated := false
	err := withRLS(ctx, r.DB, func(tx pgx.Tx) error {
		q := sqlcpg.New(tx)

		tenantUUID, err := parseUUIDText(tenant.ID)
		if err != nil {
			return err
		}
		n, err := q.UpdateTenant(ctx, sqlcpg.UpdateTenantParams{
			DisplayName:  tenant.DisplayName,
			Province:     tenant.Province,
			City:         tenant.City,
			District:     tenant.District,
			Address:      tenant.Address,
			ContactName:  tenant.ContactName,
			ContactPhone: tenant.ContactPhone,
			Remark:       tenant.Remark,
			UpdatedAt:    pgTimestamptz(tenant.UpdatedAt),
			TenantID:     tenantUUID,
		})
		if err != nil {
			return mapTenantPGError(err)
		}
		updated = n > 0
		return nil
	})
	return updated, err
}

func (r *TenantRepoPG) ToggleStatus(ctx context.Context, tenantID, status string) (bool, error) {
	updated := false
	err := withRLS(ctx, r.DB, func(tx pgx.Tx) error {
		q := sqlcpg.New(tx)

		tenantUUID, err := parseUUIDText(tenantID)
		if err != nil {
			return err
		}
		n, err := q.ToggleTenantStatus(ctx, sqlcpg.ToggleTenantStatusParams{
			Status:   status,
			TenantID: tenantUUID,
		})
		if err != nil {
			return err
		}
		if n == 0 {
			updated = false
			return nil
		}
		updated = true
		return q.UpdateUsersStatusByTenantID(ctx, sqlcpg.UpdateUsersStatusByTenantIDParams{
			Status:   status,
			TenantID: tenantUUID,
		})
	})
	return updated, err
}

func (r *TenantRepoPG) HasTenantAdmin(ctx context.Context, tenantID string) (bool, error) {
	exists := false
	err := withRLS(ctx, r.DB, func(tx pgx.Tx) error {
		q := sqlcpg.New(tx)
		tenantUUID, err := parseUUIDText(tenantID)
		if err != nil {
			return err
		}
		exists, err = q.HasTenantAdmin(ctx, tenantUUID)
		return err
	})
	return exists, err
}

func (r *TenantRepoPG) DisplayNameExists(ctx context.Context, displayName string) (bool, error) {
	exists := false
	err := withRLS(ctx, r.DB, func(tx pgx.Tx) error {
		q := sqlcpg.New(tx)
		var err error
		exists, err = q.DisplayNameExists(ctx, displayName)
		return err
	})
	return exists, err
}

func (r *TenantRepoPG) TenantAdminAccountExists(ctx context.Context, account string) (bool, error) {
	exists := false
	err := withRLS(ctx, r.DB, func(tx pgx.Tx) error {
		q := sqlcpg.New(tx)
		var err error
		exists, err = q.TenantAdminAccountExists(ctx, account)
		return err
	})
	return exists, err
}

func (r *TenantRepoPG) TenantAdminPhoneExists(ctx context.Context, adminPhone string) (bool, error) {
	exists := false
	err := withRLS(ctx, r.DB, func(tx pgx.Tx) error {
		q := sqlcpg.New(tx)
		var err error
		exists, err = q.TenantAdminPhoneExists(ctx, pgtype.Text{String: adminPhone, Valid: true})
		return err
	})
	return exists, err
}

func (r *TenantRepoPG) GetTenantAdminByTenantID(ctx context.Context, tenantID string) (*model.User, string, error) {
	var outUser *model.User
	var outAccount string
	err := withRLS(ctx, r.DB, func(tx pgx.Tx) error {
		q := sqlcpg.New(tx)
		tenantUUID, err := parseUUIDText(tenantID)
		if err != nil {
			return err
		}
		row, err := q.GetTenantAdminByTenantID(ctx, tenantUUID)
		if err != nil {
			return mapRowNotFound(err)
		}
		outAccount = row.Account
		outUser = &model.User{
			ID:                 row.UserID,
			TenantID:           toString(row.TenantID),
			Name:               row.Name,
			Phone:              row.Phone,
			Role:               row.Role,
			ScopeType:          row.ScopeType,
			TokenVersion:       int(row.TokenVersion),
			MustChangePassword: row.MustChangePassword,
			PasswordUpdatedAt:  pgTimestamptzPtr(row.PasswordUpdatedAt),
		}
		return nil
	})
	if err != nil {
		return nil, "", err
	}
	return outUser, outAccount, nil
}

func (r *TenantRepoPG) GetTenantAdminByUserID(ctx context.Context, adminUserID string) (*model.User, string, error) {
	var outUser *model.User
	var outAccount string
	err := withRLS(ctx, r.DB, func(tx pgx.Tx) error {
		q := sqlcpg.New(tx)
		adminUUID, err := parseUUIDText(adminUserID)
		if err != nil {
			return err
		}
		row, err := q.GetTenantAdminByUserID(ctx, adminUUID)
		if err != nil {
			return mapRowNotFound(err)
		}
		outAccount = row.Account
		outUser = &model.User{
			ID:                 row.UserID,
			TenantID:           toString(row.TenantID),
			Name:               row.Name,
			Phone:              row.Phone,
			Role:               row.Role,
			ScopeType:          row.ScopeType,
			TokenVersion:       int(row.TokenVersion),
			MustChangePassword: row.MustChangePassword,
			PasswordUpdatedAt:  pgTimestamptzPtr(row.PasswordUpdatedAt),
		}
		return nil
	})
	if err != nil {
		return nil, "", err
	}
	return outUser, outAccount, nil
}

func (r *TenantRepoPG) UpdateTenantAdminIdentity(ctx context.Context, adminUserID, adminAccount, adminName, adminPhone string) error {
	return withRLS(ctx, r.DB, func(tx pgx.Tx) error {
		q := sqlcpg.New(tx)
		adminUUID, err := parseUUIDText(adminUserID)
		if err != nil {
			return err
		}
		if err := q.UpdateTenantAdminProfile(ctx, sqlcpg.UpdateTenantAdminProfileParams{
			Name:   adminName,
			Phone:  adminPhone,
			UserID: adminUUID,
		}); err != nil {
			return mapTenantPGError(err)
		}
		if err := q.UpdateTenantAdminAccount(ctx, sqlcpg.UpdateTenantAdminAccountParams{
			Account: adminAccount,
			UserID:  adminUUID,
		}); err != nil {
			return mapTenantPGError(err)
		}
		return nil
	})
}

func (r *TenantRepoPG) ResetTenantAdminPassword(ctx context.Context, adminUserID, passwordHash string) error {
	return withRLS(ctx, r.DB, func(tx pgx.Tx) error {
		q := sqlcpg.New(tx)
		adminUUID, err := parseUUIDText(adminUserID)
		if err != nil {
			return err
		}
		if err := q.UpdateTenantAdminPassword(ctx, sqlcpg.UpdateTenantAdminPasswordParams{
			PasswordHash: passwordHash,
			UserID:       adminUUID,
		}); err != nil {
			return err
		}
		return q.MarkTenantAdminPasswordReset(ctx, adminUUID)
	})
}

func parseUUIDText(v string) (pgtype.UUID, error) {
	var id pgtype.UUID
	if err := id.Scan(v); err != nil {
		return pgtype.UUID{}, err
	}
	return id, nil
}

func pgTimestamptz(t time.Time) pgtype.Timestamptz {
	return pgtype.Timestamptz{Time: t, Valid: true}
}

func pgTimestamptzPtr(v pgtype.Timestamptz) *time.Time {
	if !v.Valid {
		return nil
	}
	t := v.Time
	return &t
}

func toString(v interface{}) string {
	switch x := v.(type) {
	case nil:
		return ""
	case string:
		return x
	case []byte:
		return string(x)
	default:
		return fmt.Sprint(x)
	}
}

func mapTenantPGError(err error) error {
	var pgErr *pgconn.PgError
	if !errors.As(err, &pgErr) {
		return err
	}
	if pgErr.Code != "23505" {
		return err
	}
	switch pgErr.ConstraintName {
	case "uq_tenants_display_name":
		return domainErr.ErrTenantDisplayNameExists
	case "uq_user_credentials_account", "user_credentials_account_key":
		return domainErr.ErrTenantAdminAccountExists
	case "uq_users_phone_not_null", "users_phone_key":
		return domainErr.ErrTenantAdminPhoneExists
	default:
		return domainErr.ErrConflict
	}
}
