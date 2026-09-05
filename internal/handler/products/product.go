package products

import (
	"github.com/jackc/pgx/v5/pgxpool"
	"go.uber.org/zap"
)

type ProductHandler struct {
	db *pgxpool.Pool
	logger *zap.Logger
}

func NewProductHandler(db *pgxpool.Pool,logger *zap.Logger)*ProductHandler{
	return &ProductHandler{
		db: db,
		logger: logger,
	}
}
