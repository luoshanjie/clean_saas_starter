package pg

import (
	"context"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/jackc/pgx/v5/pgxpool"

	"service/internal/domain/model"
	"service/internal/repo/pg/sqlcpg"
)

type AuthRepoPG struct {
	DB *pgxpool.Pool
}

func (r *AuthRepoPG) GetUserByAccount(ctx context.Context, account string) (*model.User, string, error) {
	var outUser *model.User
	var outHash string
	err := withRLS(ctx, r.DB, func(tx pgx.Tx) error {
		q := sqlcpg.New(tx)
		row, err := q.AuthGetUserByAccount(ctx, account)
		if err != nil {
			return err
		}
		outUser = &model.User{
			ID:                 row.ID,
			TenantID:           toString(row.TenantID),
			TenantName:         row.TenantName,
			Name:               row.Name,
			Phone:              row.Phone,
			Role:               row.Role,
			ScopeType:          row.ScopeType,
			TokenVersion:       int(row.TokenVersion),
			MustChangePassword: row.MustChangePassword,
			PasswordUpdatedAt:  pgTimestamptzPtr(row.PasswordUpdatedAt),
		}
		outHash = row.PasswordHash
		return nil
	})
	if err != nil {
		return nil, "", err
	}
	return outUser, outHash, nil
}

func (r *AuthRepoPG) GetTokenVersionByUserID(ctx context.Context, userID string) (int, error) {
	var version int
	err := withRLS(ctx, r.DB, func(tx pgx.Tx) error {
		q := sqlcpg.New(tx)
		uid, err := parseUUIDText(userID)
		if err != nil {
			return err
		}
		v, err := q.AuthGetTokenVersionByUserID(ctx, uid)
		if err != nil {
			return err
		}
		version = int(v)
		return nil
	})
	if err != nil {
		return 0, err
	}
	return version, nil
}

func (r *AuthRepoPG) CreateLoginChallenge(ctx context.Context, challenge *model.LoginChallenge) error {
	return withRLS(ctx, r.DB, func(tx pgx.Tx) error {
		q := sqlcpg.New(tx)
		id, err := parseUUIDText(challenge.ID)
		if err != nil {
			return err
		}
		userID, err := parseUUIDText(challenge.UserID)
		if err != nil {
			return err
		}
		return q.AuthCreateLoginChallenge(ctx, sqlcpg.AuthCreateLoginChallengeParams{
			ID:         id,
			UserID:     userID,
			OtpHash:    challenge.OTPHash,
			ExpiresAt:  pgTimestamptz(challenge.ExpiresAt),
			Attempts:   int32(challenge.Attempts),
			VerifiedAt: pgNullableTimestamptz(challenge.VerifiedAt),
			CreatedAt:  pgTimestamptz(challenge.CreatedAt),
		})
	})
}

func (r *AuthRepoPG) GetLoginChallengeByID(ctx context.Context, challengeID string) (*model.LoginChallenge, error) {
	var out *model.LoginChallenge
	err := withRLS(ctx, r.DB, func(tx pgx.Tx) error {
		q := sqlcpg.New(tx)
		id, err := parseUUIDText(challengeID)
		if err != nil {
			return err
		}
		row, err := q.AuthGetLoginChallengeByID(ctx, id)
		if err != nil {
			return err
		}
		out = &model.LoginChallenge{
			ID:         row.ID,
			UserID:     row.UserID,
			OTPHash:    row.OtpHash,
			ExpiresAt:  row.ExpiresAt.Time,
			Attempts:   int(row.Attempts),
			VerifiedAt: pgTimestamptzPtr(row.VerifiedAt),
			CreatedAt:  row.CreatedAt.Time,
		}
		return nil
	})
	return out, err
}

func (r *AuthRepoPG) IncreaseLoginChallengeAttempts(ctx context.Context, challengeID string) error {
	return withRLS(ctx, r.DB, func(tx pgx.Tx) error {
		q := sqlcpg.New(tx)
		id, err := parseUUIDText(challengeID)
		if err != nil {
			return err
		}
		return q.AuthIncreaseLoginChallengeAttempts(ctx, id)
	})
}

func (r *AuthRepoPG) MarkLoginChallengeVerified(ctx context.Context, challengeID string) error {
	return withRLS(ctx, r.DB, func(tx pgx.Tx) error {
		q := sqlcpg.New(tx)
		id, err := parseUUIDText(challengeID)
		if err != nil {
			return err
		}
		return q.AuthMarkLoginChallengeVerified(ctx, id)
	})
}

func (r *AuthRepoPG) SavePhoneChangeChallenge(ctx context.Context, challenge *model.PhoneChangeChallenge) error {
	return withRLS(ctx, r.DB, func(tx pgx.Tx) error {
		q := sqlcpg.New(tx)
		userID, err := parseUUIDText(challenge.UserID)
		if err != nil {
			return err
		}
		if err := q.AuthDeleteUnverifiedPhoneChangeChallengesByUserID(ctx, userID); err != nil {
			return err
		}
		id, err := parseUUIDText(challenge.ID)
		if err != nil {
			return err
		}
		return q.AuthCreatePhoneChangeChallenge(ctx, sqlcpg.AuthCreatePhoneChangeChallengeParams{
			ID:          id,
			UserID:      userID,
			NewPhone:    challenge.NewPhone,
			OtpHash:     challenge.OTPHash,
			ExpiresAt:   pgTimestamptz(challenge.ExpiresAt),
			Attempts:    int32(challenge.Attempts),
			ResendCount: int32(challenge.ResendCount),
			LastSentAt:  pgTimestamptz(challenge.LastSentAt),
			VerifiedAt:  pgNullableTimestamptz(challenge.VerifiedAt),
			CreatedAt:   pgTimestamptz(challenge.CreatedAt),
		})
	})
}

func (r *AuthRepoPG) GetPhoneChangeChallengeByID(ctx context.Context, challengeID string) (*model.PhoneChangeChallenge, error) {
	var out *model.PhoneChangeChallenge
	err := withRLS(ctx, r.DB, func(tx pgx.Tx) error {
		q := sqlcpg.New(tx)
		id, err := parseUUIDText(challengeID)
		if err != nil {
			return err
		}
		row, err := q.AuthGetPhoneChangeChallengeByID(ctx, id)
		if err != nil {
			return err
		}
		out = &model.PhoneChangeChallenge{
			ID:          row.ID,
			UserID:      row.UserID,
			NewPhone:    row.NewPhone,
			OTPHash:     row.OtpHash,
			ExpiresAt:   row.ExpiresAt.Time,
			Attempts:    int(row.Attempts),
			ResendCount: int(row.ResendCount),
			LastSentAt:  row.LastSentAt.Time,
			VerifiedAt:  pgTimestamptzPtr(row.VerifiedAt),
			CreatedAt:   row.CreatedAt.Time,
		}
		return nil
	})
	return out, err
}

func (r *AuthRepoPG) IncreasePhoneChangeChallengeAttempts(ctx context.Context, challengeID string) error {
	return withRLS(ctx, r.DB, func(tx pgx.Tx) error {
		q := sqlcpg.New(tx)
		id, err := parseUUIDText(challengeID)
		if err != nil {
			return err
		}
		return q.AuthIncreasePhoneChangeChallengeAttempts(ctx, id)
	})
}

func (r *AuthRepoPG) MarkPhoneChangeChallengeVerified(ctx context.Context, challengeID string) error {
	return withRLS(ctx, r.DB, func(tx pgx.Tx) error {
		q := sqlcpg.New(tx)
		id, err := parseUUIDText(challengeID)
		if err != nil {
			return err
		}
		return q.AuthMarkPhoneChangeChallengeVerified(ctx, id)
	})
}

func (r *AuthRepoPG) GetUserByID(ctx context.Context, userID string) (*model.User, error) {
	var out *model.User
	err := withRLS(ctx, r.DB, func(tx pgx.Tx) error {
		q := sqlcpg.New(tx)
		uid, err := parseUUIDText(userID)
		if err != nil {
			return err
		}
		row, err := q.AuthGetUserByID(ctx, uid)
		if err != nil {
			return err
		}
		out = &model.User{
			ID:                 row.ID,
			TenantID:           toString(row.TenantID),
			TenantName:         row.TenantName,
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
	return out, err
}

func (r *AuthRepoPG) GetPasswordHashByUserID(ctx context.Context, userID string) (string, error) {
	var hash string
	err := withRLS(ctx, r.DB, func(tx pgx.Tx) error {
		q := sqlcpg.New(tx)
		uid, err := parseUUIDText(userID)
		if err != nil {
			return err
		}
		v, err := q.AuthGetPasswordHashByUserID(ctx, uid)
		if err != nil {
			return err
		}
		hash = v
		return nil
	})
	return hash, err
}

func (r *AuthRepoPG) UpdatePasswordByUserID(ctx context.Context, userID, passwordHash string, mustChange bool) error {
	return withRLS(ctx, r.DB, func(tx pgx.Tx) error {
		q := sqlcpg.New(tx)
		uid, err := parseUUIDText(userID)
		if err != nil {
			return err
		}
		if err := q.AuthUpdatePasswordHashByUserID(ctx, sqlcpg.AuthUpdatePasswordHashByUserIDParams{
			PasswordHash: passwordHash,
			UserID:       uid,
		}); err != nil {
			return err
		}
		return q.AuthUpdatePasswordMetaByUserID(ctx, sqlcpg.AuthUpdatePasswordMetaByUserIDParams{
			MustChangePassword: mustChange,
			UserID:             uid,
		})
	})
}

func (r *AuthRepoPG) PhoneExists(ctx context.Context, phone string) (bool, error) {
	var exists bool
	err := withRLS(ctx, r.DB, func(tx pgx.Tx) error {
		q := sqlcpg.New(tx)
		v, err := q.AuthPhoneExists(ctx, pgtype.Text{String: phone, Valid: true})
		if err != nil {
			return err
		}
		exists = v
		return nil
	})
	return exists, err
}

func (r *AuthRepoPG) UpdatePhoneByUserID(ctx context.Context, userID, phone string) error {
	return withRLS(ctx, r.DB, func(tx pgx.Tx) error {
		q := sqlcpg.New(tx)
		uid, err := parseUUIDText(userID)
		if err != nil {
			return err
		}
		return q.AuthUpdatePhoneByUserID(ctx, sqlcpg.AuthUpdatePhoneByUserIDParams{
			Phone:  pgtype.Text{String: phone, Valid: true},
			UserID: uid,
		})
	})
}

func (r *AuthRepoPG) UpdateDisplayNameByUserID(ctx context.Context, userID, name string) error {
	return withRLS(ctx, r.DB, func(tx pgx.Tx) error {
		q := sqlcpg.New(tx)
		uid, err := parseUUIDText(userID)
		if err != nil {
			return err
		}
		return q.AuthUpdateDisplayNameByUserID(ctx, sqlcpg.AuthUpdateDisplayNameByUserIDParams{
			Name:   name,
			UserID: uid,
		})
	})
}

func pgNullableTimestamptz(t *time.Time) pgtype.Timestamptz {
	if t == nil {
		return pgtype.Timestamptz{}
	}
	return pgTimestamptz(*t)
}
