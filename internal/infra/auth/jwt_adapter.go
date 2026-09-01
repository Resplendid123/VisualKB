package auth

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"time"

	"learn/internal/domain/auth"
	"learn/internal/infra/config"

	"github.com/golang-jwt/jwt/v5"
)

type jwtAdapter struct {
	cfg config.AuthConfig
}

func NewJWTAdapter(cfg config.AuthConfig) auth.AuthRepo {
	return &jwtAdapter{cfg: cfg}
}

func (a *jwtAdapter) Issue(userID int64) (string, error) {
	claims := jwt.MapClaims{
		"sub": fmt.Sprint(userID),
		"exp": time.Now().Add(a.cfg.TTL).Unix(),
		"iat": time.Now().Unix(),
	}
	if a.cfg.Issuer != "" {
		claims["iss"] = a.cfg.Issuer
	}
	return jwt.NewWithClaims(jwt.SigningMethodHS256, claims).SignedString([]byte(a.cfg.JWTSecret))
}

func (a *jwtAdapter) Verify(_ context.Context, raw string) (auth.Identity, error) {
	identity, expired, err := a.parse(raw, true)
	if err != nil {
		return auth.Identity{}, err
	}
	if expired {
		return auth.Identity{}, fmt.Errorf("token expired")
	}
	return identity, nil
}

func (a *jwtAdapter) VerifyForRefresh(ctx context.Context, raw string) (auth.Identity, error) {
	identity, expired, err := a.parse(raw, false)
	if err != nil {
		return auth.Identity{}, err
	}
	if expired {
		slog.DebugContext(ctx, "token expired, but within refresh window")
	}
	return identity, nil
}

func (a *jwtAdapter) parse(raw string, requireValid bool) (auth.Identity, bool, error) {
	token, err := jwt.Parse(raw, func(token *jwt.Token) (any, error) {
		if token.Method != jwt.SigningMethodHS256 {
			return nil, fmt.Errorf("unexpected signing method")
		}
		return []byte(a.cfg.JWTSecret), nil
	})

	expired := errors.Is(err, jwt.ErrTokenExpired)

	if err != nil && !expired {
		return auth.Identity{}, false, fmt.Errorf("invalid token")
	}
	if token == nil {
		return auth.Identity{}, false, fmt.Errorf("invalid token")
	}

	claims, ok := token.Claims.(jwt.MapClaims)
	if !ok {
		return auth.Identity{}, false, fmt.Errorf("invalid claims")
	}

	expFloat, ok := claims["exp"].(float64)
	if !ok {
		return auth.Identity{}, false, fmt.Errorf("invalid exp")
	}
	expiry := time.Unix(int64(expFloat), 0)
	now := time.Now()

	if requireValid && now.After(expiry) {
		return auth.Identity{}, true, fmt.Errorf("token expired")
	}
	if expired && now.Sub(expiry) > a.cfg.RefreshWindow {
		return auth.Identity{}, true, fmt.Errorf("refresh window exceeded")
	}

	sub, ok := claims["sub"].(string)
	if !ok {
		return auth.Identity{}, false, fmt.Errorf("invalid subject")
	}
	var id int64
	if _, err := fmt.Sscan(sub, &id); err != nil {
		return auth.Identity{}, false, fmt.Errorf("invalid subject")
	}
	return auth.Identity{UserID: id}, expired, nil
}
