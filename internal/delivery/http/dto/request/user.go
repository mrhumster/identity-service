package request

type UserRequest struct {
	Password string `json:"password" binding:"required,min=6,max=100"`
	Email    string `json:"email" binding:"required,email"`
}

type UpdateUserRequest struct {
	Email string `json:"email,omitempty" binding:"omitempty,email"`
}
