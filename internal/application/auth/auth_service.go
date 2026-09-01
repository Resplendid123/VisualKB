package auth

import (
	"context"
	"errors"
	"learn/internal/domain"
	"learn/internal/domain/auth"
	domainuser "learn/internal/domain/user"

	"golang.org/x/crypto/bcrypt"
)

type AuthService struct {
	userRepo domainuser.UserRepo
	authRepo auth.AuthRepo
}

func NewAuthService(userRepo domainuser.UserRepo, authRepo auth.AuthRepo) *AuthService {
	return &AuthService{userRepo: userRepo, authRepo: authRepo}
}

// Register creates user and issues token.
func (s *AuthService) Register(ctx context.Context, name, email, password string) (*domainuser.User, string, error) {
	u := &domainuser.User{Name: name, Email: email}
	if err := u.Validate(); err != nil {
		return nil, "", err
	}

	hash, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		return nil, "", domain.ErrUserCreateFailed
	}
	u.PasswordHash = string(hash)

	if err := s.userRepo.Create(ctx, u); err != nil {
		if errors.Is(err, domain.ErrEmailTaken) {
			return nil, "", domain.ErrEmailTaken
		}
		return nil, "", domain.ErrUserCreateFailed
	}

	token, err := s.authRepo.Issue(u.ID)
	if err != nil {
		return nil, "", domain.ErrUserCreateFailed
	}
	return u, token, nil
}

// Login verifies credentials and issues token.
func (s *AuthService) Login(ctx context.Context, email, password string) (*domainuser.User, string, error) {
	u, err := s.userRepo.FindByEmail(ctx, email)
	if err != nil {
		if !errors.Is(err, domain.ErrUserNotFound) {
			return nil, "", domain.ErrUserNotFound
		}
		return nil, "", domain.ErrInvalidCredentials
	}
	if bcrypt.CompareHashAndPassword([]byte(u.PasswordHash), []byte(password)) != nil {
		return nil, "", domain.ErrInvalidCredentials
	}
	token, err := s.authRepo.Issue(u.ID)
	if err != nil {
		return nil, "", domain.ErrInvalidCredentials
	}
	return u, token, nil
}

func (s *AuthService) Refresh(ctx context.Context, rawToken string) (*domainuser.User, string, error) {
	identity, err := s.authRepo.VerifyForRefresh(ctx, rawToken)
	if err != nil {
		return nil, "", domain.ErrTokenRefreshFailed
	}
	u, err := s.userRepo.FindByID(ctx, identity.UserID)
	if err != nil {
		return nil, "", domain.ErrTokenRefreshFailed
	}
	token, err := s.authRepo.Issue(u.ID)
	if err != nil {
		return nil, "", domain.ErrTokenRefreshFailed
	}
	return u, token, nil
}

func (s *AuthService) VerifyToken(ctx context.Context, token string) (*auth.Identity, error) {
	identity, err := s.authRepo.Verify(ctx, token)
	if err != nil {
		return nil, err
	}
	if _, err := s.userRepo.FindByID(ctx, identity.UserID); err != nil {
		return nil, domain.ErrTokenInvalid
	}
	return &identity, nil
}
