package handler

import (

	"github.com/jackc/pgx/v5/pgxpool"
)

type ProductHandler struct {
	db *pgxpool.Pool
}

func NewProductHandler(db *pgxpool.Pool)*ProductHandler{
	return &ProductHandler{
		db: db,
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