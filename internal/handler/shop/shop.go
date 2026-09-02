package shop

import (
	"github.com/jackc/pgx/v5/pgxpool"
	"go.uber.org/zap"
)

type ShopHandler struct {
	db *pgxpool.Pool
	logger *zap.Logger
}

func NewShopHandler(db *pgxpool.Pool,logger *zap.Logger)*ShopHandler{
	return &ShopHandler{
		db: db,
		logger: logger,
	}
}