package dto

type RegisterUserRequest struct {
	Name     string `json:"name" required:"true" validate:"min=3,max=20"`
	Email    string `json:"email" required:"true" validate:"email"`
	Password string `json:"password" required:"true" validate:"min=5,max=20"`
}

type LoginUserRequest struct {
	Email    string `json:"email" required:"true" validate:"email"`
	Password string `json:"password" required:"true"`
}
