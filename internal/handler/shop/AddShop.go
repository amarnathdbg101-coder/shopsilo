package shop

import "github.com/jackc/pgx/v5/pgxpool"

type Address struct {
	Village  string `json:"village"`
	Post     string `json:"post"`
	District string `json:"district"`
	State    string `json:"state"`
	Pincode  string `json:"pincode"`
	Landmark string `json:"landmark"`
}

type Shop struct {
	Id       int      `json:"id"`
	Name     string   `json:"name"`
	Address  Address  `json:"address"`
	Category []string `json:"category"`
	UserId   int      `json:"userid"`
}

type ShopHandler struct {
	db *pgxpool.Pool
}

func NewShopHandler(db *pgxpool.Pool) *ShopHandler {
	return &ShopHandler{
		db: db,
	}
}
