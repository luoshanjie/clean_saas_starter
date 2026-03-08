package sms

import "context"

// MockSMSSender 用于开发联调；生产环境请替换为真实短信网关适配器。
type MockSMSSender struct{}

func (m MockSMSSender) SendLoginOTP(ctx context.Context, phone, code string) error {
	return nil
}

func (m MockSMSSender) SendChangePhoneOTP(ctx context.Context, phone, code string) error {
	return nil
}
