package authctx

import "context"

type Info struct {
	UserID    string
	TenantID  string
	ScopeType string
}

type ctxKey struct{}

func With(ctx context.Context, info Info) context.Context {
	return context.WithValue(ctx, ctxKey{}, info)
}

func From(ctx context.Context) (Info, bool) {
	v := ctx.Value(ctxKey{})
	if v == nil {
		return Info{}, false
	}
	info, ok := v.(Info)
	return info, ok
}
