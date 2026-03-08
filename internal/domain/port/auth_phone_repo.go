package port

import "context"

type AuthPhoneRepo interface {
	PhoneExists(ctx context.Context, phone string) (bool, error)
	UpdatePhoneByUserID(ctx context.Context, userID, phone string) error
}
