package shop

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"shopMe/internal/middleware"
	"shopMe/internal/reuse"

	"github.com/go-playground/validator/v10"
)

type input struct {
	ShopName  string  `json:"name" validate:"required,min=3"`
	Timing    string  `json:"timing" validate:"required"`
	Category  string  `json:"category" validate:"required"`
	Village   string  `json:"village" validate:"required"`
	Post      string  `json:"post"`
	District  string  `json:"district" validate:"required"`
	State     string  `json:"state" validate:"required"`
	Pincode   string  `json:"pincode" validate:"required,len=6,numeric"`
	Landmark  string  `json:"landmark" validate:"required"`
	Latitude  float64 `json:"latitude" validate:"required"`
	Longitude float64 `json:"longitude" validate:"required"`
}

func (sh *ShopHandler) CreateShop(w http.ResponseWriter, r *http.Request) {
	userId := r.Context().Value(middleware.UserIDContextKey).(int)
	var input input
	err := json.NewDecoder(r.Body).Decode(&input)
	if err != nil {
		reuse.Error(w, http.StatusBadRequest, reuse.ErrInvalidInput, "invalid input")
		return
	}
	// Run validation
	err = reuse.Validate.Struct(input)
	if err != nil {
		for _, e := range err.(validator.ValidationErrors) {
			reuse.Error(w, http.StatusBadRequest, reuse.ErrInvalidInput,
				fmt.Sprintf("Failed %s on %s len", e.Field(), e.Tag()))
			return
		}
	}
	// insert address
	var addressId int
	query1 := `insert into addresses (village,post,district,state,pincode,landmark,latitude,longitude) values ($1,$2,$3,$4,$5,$6,$7,$8) returning id`
	err = sh.db.QueryRow(r.Context(), query1, input.Village, input.Post, input.District, input.State, input.Pincode, input.Landmark, input.Latitude, input.Longitude).Scan(&addressId)
	if err != nil {
		log.Println(err.Error())
		reuse.Error(w, http.StatusInternalServerError, reuse.ErrInternal, "address insert failed")
		return
	}

	// insert shop
	var shopId int
	query2 := `insert into shops (user_id,name,address_id,timing,category) values ($1,$2,$3,$4,$5) returning id`
	err = sh.db.QueryRow(r.Context(), query2, userId, input.ShopName, addressId, input.Timing, input.Category).Scan(&shopId)
	if err != nil {
		log.Println(err.Error())
		reuse.Error(w, http.StatusInternalServerError, reuse.ErrDBFailure, "User have already shop")
		return
	}

	reuse.Success(w, "Shop created successfully", map[string]any{
		"shopID":   shopId,
		"shopName": input.ShopName,
		"timing":   input.Timing,
		"category": input.Category,
	})
}
