package port

import "context"

type LoginOTPSender interface {
	SendLoginOTP(ctx context.Context, phone, code string) error
}

type ChangePhoneOTPSender interface {
	SendChangePhoneOTP(ctx context.Context, phone, code string) error
}
