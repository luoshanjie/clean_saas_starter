package middleware

import (
	"crypto/rand"
	"encoding/hex"

	"github.com/labstack/echo/v4"
)

const CtxRequestID = "request_id"

// RequestIDMiddleware generates or propagates request_id.
func RequestIDMiddleware() echo.MiddlewareFunc {
	return func(next echo.HandlerFunc) echo.HandlerFunc {
		return func(c echo.Context) error {
			id := c.Request().Header.Get(echo.HeaderXRequestID)
			if id == "" {
				id = newRequestID()
			}
			c.Set(CtxRequestID, id)
			c.Response().Header().Set(echo.HeaderXRequestID, id)
			return next(c)
		}
	}
}

func GetRequestID(c echo.Context) string {
	v, _ := c.Get(CtxRequestID).(string)
	return v
}

func newRequestID() string {
	var b [16]byte
	_, _ = rand.Read(b[:])
	return hex.EncodeToString(b[:])
}
