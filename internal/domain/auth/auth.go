package auth

import "context"

type Identity struct {
	UserID int64
}

type AuthRepo interface {
	Issue(userID int64) (string, error)
	Verify(ctx context.Context, token string) (Identity, error)
	VerifyForRefresh(ctx context.Context, raw string) (Identity, error)
}
