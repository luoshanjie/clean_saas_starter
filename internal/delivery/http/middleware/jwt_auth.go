package middleware

import (
	"net/http"
	"strings"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/labstack/echo/v4"

	"service/internal/delivery/http/resp"
)

type JWTMiddleware struct {
	Secret []byte
}

type Claims struct {
	UserID       string `json:"user_id"`
	Name         string `json:"name"`
	Role         string `json:"role"`
	ScopeType    string `json:"scope_type"`
	TenantID     string `json:"tenant_id"`
	TokenType    string `json:"token_type"`
	TokenVersion int    `json:"token_version"`
	jwt.RegisteredClaims
}

func (m JWTMiddleware) ParseToken(tokenStr string) (*Claims, error) {
	claims := &Claims{}
	_, err := jwt.ParseWithClaims(tokenStr, claims, func(token *jwt.Token) (any, error) {
		return m.Secret, nil
	})
	if err != nil {
		return nil, err
	}
	if claims.ExpiresAt != nil && time.Until(claims.ExpiresAt.Time) <= 0 {
		return nil, jwt.ErrTokenExpired
	}
	return claims, nil
}

func (m JWTMiddleware) MiddlewareFunc(next echo.HandlerFunc) echo.HandlerFunc {
	return func(c echo.Context) error {
		auth := c.Request().Header.Get("Authorization")
		if auth == "" {
			return c.JSON(http.StatusOK, resp.ErrorWithRequestID(GetRequestID(c), resp.CodeUnauthorized, "unauthorized"))
		}
		parts := strings.SplitN(auth, " ", 2)
		if len(parts) != 2 || !strings.EqualFold(parts[0], "Bearer") {
			return c.JSON(http.StatusOK, resp.ErrorWithRequestID(GetRequestID(c), resp.CodeUnauthorized, "unauthorized"))
		}

		claims, err := m.ParseToken(parts[1])
		if err != nil {
			return c.JSON(http.StatusOK, resp.ErrorWithRequestID(GetRequestID(c), resp.CodeUnauthorized, "unauthorized"))
		}
		if claims.TokenType == "refresh" {
			return c.JSON(http.StatusOK, resp.ErrorWithRequestID(GetRequestID(c), resp.CodeUnauthorized, "unauthorized"))
		}

		c.Set(CtxUserID, claims.UserID)
		c.Set(CtxUserName, claims.Name)
		c.Set(CtxRole, claims.Role)
		c.Set(CtxScopeType, claims.ScopeType)
		c.Set(CtxTenantID, claims.TenantID)
		c.Set(CtxTokenVersion, claims.TokenVersion)

		return next(c)
	}
}
