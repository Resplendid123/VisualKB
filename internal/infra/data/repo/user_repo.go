package repo

import (
	"context"
	"errors"
	"fmt"

	"learn/internal/domain"
	domainuser "learn/internal/domain/user"
	"learn/internal/infra/data/model"

	"gorm.io/gorm"
)

type userRepo struct {
	db *gorm.DB
}

func NewUserRepo(db *gorm.DB) domainuser.UserRepo {
	return &userRepo{db: db}
}

func (r *userRepo) Create(ctx context.Context, u *domainuser.User) error {

	m := userToModel(u)
	if err := r.db.WithContext(ctx).Create(m).Error; err != nil {
		if errors.Is(err, gorm.ErrDuplicatedKey) {
			return fmt.Errorf("%w: email=%s", domain.ErrEmailTaken, u.Email)
		}
		return err
	}
	u.ID = m.ID
	return nil
}

func (r *userRepo) FindByID(ctx context.Context, id int64) (*domainuser.User, error) {
	var m model.User
	if err := r.db.WithContext(ctx).First(&m, id).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, domain.ErrUserNotFound
		}
		return nil, err
	}
	return userToDomain(&m), nil
}

func (r *userRepo) FindByEmail(ctx context.Context, email string) (*domainuser.User, error) {
	var m model.User
	err := r.db.WithContext(ctx).Where("email = ?", email).First(&m).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, domain.ErrUserNotFound
		}
		return nil, err
	}
	return userToDomain(&m), nil
}

func (r *userRepo) UpdateImmutable(ctx context.Context, userID int64, immutable string) error {
	res := r.db.WithContext(ctx).
		Model(&model.User{}).
		Where("id = ?", userID).
		Updates(map[string]any{"immutable": immutable})
	if res.Error != nil {
		return res.Error
	}
	if res.RowsAffected == 0 {
		return domain.ErrUserNotFound
	}
	return nil
}

func (r *userRepo) UpdateMutable(ctx context.Context, userID int64, mutable string) error {
	res := r.db.WithContext(ctx).
		Model(&model.User{}).
		Where("id = ?", userID).
		Updates(map[string]any{"mutable": mutable})
	if res.Error != nil {
		return res.Error
	}
	if res.RowsAffected == 0 {
		return domain.ErrUserNotFound
	}
	return nil
}

func userToDomain(m *model.User) *domainuser.User {
	return &domainuser.User{
		ID:           m.ID,
		Name:         m.Name,
		Email:        m.Email,
		PasswordHash: m.PasswordHash,
		Immutable:    m.Immutable,
		Mutable:      m.Mutable,
		CreatedAt:    m.CreatedAt,
		UpdatedAt:    m.UpdatedAt,
	}
}

func userToModel(u *domainuser.User) *model.User {
	return &model.User{
		ID:           u.ID,
		Name:         u.Name,
		Email:        u.Email,
		PasswordHash: u.PasswordHash,
		Immutable:    u.Immutable,
		Mutable:      u.Mutable,
		CreatedAt:    u.CreatedAt,
		UpdatedAt:    u.UpdatedAt,
	}
}

var _ domainuser.UserRepo = (*userRepo)(nil)
