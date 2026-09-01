package user

import (
	"context"
	"strings"
	"time"

	"learn/internal/domain"
)

type User struct {
	ID           int64
	Name         string
	Email        string
	PasswordHash string
	Immutable    string
	Mutable      string
	CreatedAt    time.Time
	UpdatedAt    time.Time
}

func (u User) Validate() error {
	if strings.TrimSpace(u.Name) == "" {
		return domain.ErrInvalidName
	}
	if !isValidEmail(u.Email) {
		return domain.ErrInvalidEmail
	}
	return nil
}

func isValidEmail(email string) bool {
	email = strings.TrimSpace(email)
	if email == "" {
		return false
	}
	at := strings.Index(email, "@")
	if at <= 0 || at == len(email)-1 {
		return false
	}
	if strings.Index(email[at+1:], "@") >= 0 {
		return false
	}
	return strings.Contains(email, ".")
}

type UserRepo interface {
	Create(ctx context.Context, user *User) error
	FindByID(ctx context.Context, id int64) (*User, error)
	FindByEmail(ctx context.Context, email string) (*User, error)
	UpdateImmutable(ctx context.Context, userID int64, immutable string) error
	UpdateMutable(ctx context.Context, userID int64, mutable string) error
}
