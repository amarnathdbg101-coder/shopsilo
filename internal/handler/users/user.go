package users

import (
	"github.com/jackc/pgx/v5/pgxpool"
	"go.uber.org/zap"
)

type UserHandler struct {
	db     *pgxpool.Pool
	logger *zap.Logger
}

func NewUserHandler(db *pgxpool.Pool, logger *zap.Logger) *UserHandler {
	return &UserHandler{
		db:     db,
		logger: logger,
	}
}
