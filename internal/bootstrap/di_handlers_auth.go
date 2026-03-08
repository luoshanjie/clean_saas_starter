package bootstrap

import (
	"context"
	"errors"
	"time"

	"github.com/golang-jwt/jwt/v5"

	"service/internal/app/usecase"
	"service/internal/delivery/http/handler"
	"service/internal/delivery/http/middleware"
	"service/internal/domain/model"
	smsrepo "service/internal/repo/sms"
)

func wireAuthHandlers(h *bootstrapHandlers, d handlerDeps) {
	tokenGen := func(ctx context.Context, user *model.User) (string, string, int, error) {
		accessExpireSec := 3600
		refreshExpireSec := 604800
		nowAt := d.now()
		refresh := middleware.Claims{
			UserID:       user.ID,
			Name:         user.Name,
			Role:         user.Role,
			ScopeType:    user.ScopeType,
			TenantID:     user.TenantID,
			TokenType:    "refresh",
			TokenVersion: user.TokenVersion,
			RegisteredClaims: jwt.RegisteredClaims{
				ExpiresAt: jwt.NewNumericDate(nowAt.Add(time.Duration(refreshExpireSec) * time.Second)),
			},
		}
		claims := middleware.Claims{
			UserID:       user.ID,
			Name:         user.Name,
			Role:         user.Role,
			ScopeType:    user.ScopeType,
			TenantID:     user.TenantID,
			TokenType:    "access",
			TokenVersion: user.TokenVersion,
			RegisteredClaims: jwt.RegisteredClaims{
				ExpiresAt: jwt.NewNumericDate(nowAt.Add(time.Duration(accessExpireSec) * time.Second)),
			},
		}
		accessToken := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
		refreshToken := jwt.NewWithClaims(jwt.SigningMethodHS256, refresh)
		a, err := accessToken.SignedString([]byte(d.jwtSecret))
		if err != nil {
			return "", "", 0, err
		}
		r, err := refreshToken.SignedString([]byte(d.jwtSecret))
		if err != nil {
			return "", "", 0, err
		}
		return a, r, accessExpireSec, nil
	}
	authLoginChallenge := &usecase.AuthLoginChallengeUsecase{
		Repo:      d.repos.authRepo,
		Challenge: d.repos.authRepo,
		Sender:    smsrepo.MockSMSSender{},
		IDGen:     d.idGen,
		Now:       d.now,
		MockCode:  "123456",
	}
	authLoginVerify := &usecase.AuthLoginVerifyUsecase{
		Challenge: d.repos.authRepo,
		TokenGen:  tokenGen,
		Now:       d.now,
	}
	authRefresh := &usecase.AuthRefreshUsecase{
		ParseRefresh: func(token string) (*model.User, error) {
			claims, err := middleware.JWTMiddleware{Secret: []byte(d.jwtSecret)}.ParseToken(token)
			if err != nil {
				return nil, err
			}
			if claims.TokenType != "refresh" {
				return nil, errors.New("invalid token type")
			}
			return &model.User{
				ID:           claims.UserID,
				Name:         claims.Name,
				Role:         claims.Role,
				ScopeType:    claims.ScopeType,
				TenantID:     claims.TenantID,
				TokenVersion: claims.TokenVersion,
			}, nil
		},
		TokenGen: tokenGen,
		Repo:     d.repos.authRepo,
	}
	authMe := &usecase.AuthMeUsecase{Perm: d.perm, AuthRepo: d.repos.authRepo}
	authChangePassword := &usecase.AuthChangePasswordUsecase{Repo: d.repos.authRepo}
	authUpdateDisplayName := &usecase.AuthUpdateDisplayNameUsecase{Repo: d.repos.authRepo}
	authChangePhoneChallenge := &usecase.AuthChangePhoneChallengeUsecase{
		AuthRepo:      d.repos.authRepo,
		PhoneRepo:     d.repos.authRepo,
		ChallengeRepo: d.repos.authRepo,
		Sender:        smsrepo.MockSMSSender{},
		IDGen:         d.idGen,
		Now:           d.now,
		MockCode:      "123456",
	}
	authChangePhoneResend := &usecase.AuthChangePhoneResendUsecase{
		ChallengeRepo: d.repos.authRepo,
		Sender:        smsrepo.MockSMSSender{},
		Now:           d.now,
		MockCode:      "123456",
	}
	authChangePhoneVerify := &usecase.AuthChangePhoneVerifyUsecase{
		AuthRepo:      d.repos.authRepo,
		PhoneRepo:     d.repos.authRepo,
		ChallengeRepo: d.repos.authRepo,
		Now:           d.now,
	}
	auditWrite := &usecase.AuditWriteUsecase{
		Repo:  d.repos.auditRepo,
		IDGen: d.idGen,
		Now:   d.now,
	}

	h.authHandler = &handler.AuthHandler{
		LoginChallengeUC:       authLoginChallenge,
		LoginVerifyUC:          authLoginVerify,
		RefreshUC:              authRefresh,
		MeUC:                   authMe,
		ChangePasswordUC:       authChangePassword,
		UpdateDisplayNameUC:    authUpdateDisplayName,
		ChangePhoneChallengeUC: authChangePhoneChallenge,
		ChangePhoneResendUC:    authChangePhoneResend,
		ChangePhoneVerifyUC:    authChangePhoneVerify,
		AuditUC:                auditWrite,
		JWT:                    middleware.JWTMiddleware{Secret: []byte(d.jwtSecret)},
	}
	h.healthHandler = &handler.HealthHandler{}
}
