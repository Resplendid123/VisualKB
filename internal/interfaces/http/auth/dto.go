package auth

type RegisterRequest struct {
	Name     string `json:"name" binding:"required,max=64"`
	Email    string `json:"email" binding:"required,email,max=128"`
	Password string `json:"password" binding:"required,min=6,max=72"`
}
type LoginRequest struct {
	Email    string `json:"email" binding:"required,email"`
	Password string `json:"password" binding:"required"`
}
type RefreshRequest struct {
	AccessToken string `json:"access_token" binding:"required"`
}

type authUserPayload struct {
	ID    int64  `json:"id"`
	Name  string `json:"name"`
	Email string `json:"email"`
}

type authResponse struct {
	AccessToken string          `json:"access_token"`
	User        authUserPayload `json:"user"`
}
