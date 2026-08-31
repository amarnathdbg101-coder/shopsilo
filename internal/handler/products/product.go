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

type ProductAddInput struct {
	ProuductId int
	Name string
	Price float64
	Title string
}

type ProductUpdateInput struct {
	Name string
	Price string
	Title string
}