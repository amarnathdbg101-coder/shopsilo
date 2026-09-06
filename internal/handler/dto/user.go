package dto

type RegisterInput struct {
	Name     string `json:"name"`
	Email    string `json:"email" validate:"required,email"`
	Password string `json:"password" validate:"required,min=6"`
}

type RegisterResponse struct {
	Name  string `json:"name"`
	Email string `json:"email"`
}


type LoginInput struct {
    Email    string `json:"email" validate:"required,email"`
    Password string `json:"password" validate:"required,min=6"`
}

type LoginResponse struct {
    Token string `json:"token"`
}
