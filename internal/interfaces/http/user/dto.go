package user

import domainuser "learn/internal/domain/user"

type CreateUserRequest struct {
	Name  string `json:"name" binding:"required,max=64"`
	Email string `json:"email" binding:"required,email,max=128"`
}

type UserResponse struct {
	ID        int64  `json:"id"`
	Name      string `json:"name"`
	Email     string `json:"email"`
	CreatedAt int64  `json:"created_at"`
	UpdatedAt int64  `json:"updated_at"`
}

type GetPortraitResponse struct {
	Immutable string `json:"immutable"`
	Mutable   string `json:"mutable"`
}

type UpdateImmutableRequest struct {
	Immutable string `json:"immutable" binding:"max=4000"`
}

type UpdateMutableRequest struct {
	Mutable string `json:"mutable" binding:"max=4000"`
}

func toUserResponse(u *domainuser.User) UserResponse {
	return UserResponse{
		ID:        u.ID,
		Name:      u.Name,
		Email:     u.Email,
		CreatedAt: u.CreatedAt.Unix(),
		UpdatedAt: u.UpdatedAt.Unix(),
	}
}

func toPortraitResponse(u *domainuser.User) GetPortraitResponse {
	return GetPortraitResponse{
		Immutable: u.Immutable,
		Mutable:   u.Mutable,
	}
}
