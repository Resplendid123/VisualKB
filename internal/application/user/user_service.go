package user

import (
	"context"
	"errors"
	"strings"

	"learn/internal/domain"
	domainuser "learn/internal/domain/user"
)

const portraitFieldMaxLen = 4000

type UserService struct {
	userRepo domainuser.UserRepo
}

func NewUserService(userRepo domainuser.UserRepo) *UserService {
	return &UserService{userRepo: userRepo}
}

// Create persists user; K8s ns lazy.
func (s *UserService) Create(ctx context.Context, name, email string) (*domainuser.User, error) {
	u := &domainuser.User{Name: name, Email: email}
	if err := u.Validate(); err != nil {
		return nil, err
	}
	if err := s.userRepo.Create(ctx, u); err != nil {
		if errors.Is(err, domain.ErrEmailTaken) {
			return nil, domain.ErrInvalidEmail
		}
		return nil, domain.ErrUserCreateFailed
	}
	return u, nil
}

// GetByID fetches user by ID.
func (s *UserService) GetByID(ctx context.Context, userID int64) (*domainuser.User, error) {
	u, err := s.userRepo.FindByID(ctx, userID)
	if err != nil {
		if errors.Is(err, domain.ErrUserNotFound) {
			return nil, domain.ErrUserNotFound
		}
		return nil, domain.ErrUserFetchFailed
	}
	return u, nil
}

func (s *UserService) GetPortrait(ctx context.Context, userID int64) (immutable, mutable string, err error) {
	u, err := s.GetByID(ctx, userID)
	if err != nil {
		return "", "", err
	}
	return u.Immutable, u.Mutable, nil
}

func (s *UserService) UpdateImmutable(ctx context.Context, userID int64, immutable string) error {
	return s.writePortrait(ctx, userID, immutable, s.userRepo.UpdateImmutable)
}

func (s *UserService) UpdateMutable(ctx context.Context, userID int64, mutable string) error {
	return s.writePortrait(ctx, userID, mutable, s.userRepo.UpdateMutable)
}

func (s *UserService) writePortrait(
	ctx context.Context,
	userID int64,
	value string,
	write func(context.Context, int64, string) error,
) error {
	trimmed := strings.TrimSpace(value)
	if len(trimmed) > portraitFieldMaxLen {
		return domain.ErrPortraitTooLong
	}
	if err := write(ctx, userID, trimmed); err != nil {
		if errors.Is(err, domain.ErrUserNotFound) {
			return domain.ErrUserNotFound
		}
		return domain.ErrUserFetchFailed
	}
	return nil
}
