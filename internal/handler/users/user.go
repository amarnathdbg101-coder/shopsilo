package users

import "github.com/jackc/pgx/v5/pgxpool"

type UserHandler struct {
	db *pgxpool.Pool
}

func NewUserHandler(db *pgxpool.Pool)*UserHandler{
	return &UserHandler{
		db: db,
	}
}

type LoginInput struct {
	Email    string
	Password string
}

type LoginOutput struct {
	UserId   int
	Name     string
	Email    string
	JwtToken string
}

type RegisterInput struct {
	UserId   int
	Name     string
	Email    string
	Password string
}
