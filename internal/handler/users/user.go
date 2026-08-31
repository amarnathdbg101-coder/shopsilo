package users

import (
	"github.com/jackc/pgx/v5/pgxpool"
	"go.uber.org/zap"
)

type UserHandler struct {
	db *pgxpool.Pool
	logger *zap.Logger
}

func NewUserHandler(db *pgxpool.Pool,logger *zap.Logger)*UserHandler{
	return &UserHandler{
		db: db,
		logger: logger,
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
