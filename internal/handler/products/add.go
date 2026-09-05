package products

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"shopMe/internal/middleware"
	"shopMe/internal/reuse"

	"github.com/go-playground/validator/v10"
)

type productInput struct {
	Name        string  `json:"name" validate:"required,min=3"`
	Title       string  `json:"title" validate:"required,min=3"`
	Price       float64 `json:"price" validate:"required,numeric"`
	Mrp         float64 `json:"mrp" validate:"required,numeric"`
	Category    string  `json:"category" validate:"required"`
	Description string  `json:"description"`
}

func (ph *ProductHandler) AddProduct(w http.ResponseWriter, r *http.Request) {
	userID := r.Context().Value(middleware.UserIDContextKey).(int)

	var input productInput
	err := json.NewDecoder(r.Body).Decode(&input)
	if err != nil {
		reuse.Error(w, http.StatusBadRequest, reuse.ErrInvalidInput, "Wrong Input")
		return
	}

	err = reuse.Validate.Struct(input)
	if err != nil {
		for _, e := range err.(validator.ValidationErrors) {
			reuse.Error(w, http.StatusBadRequest, reuse.ErrInvalidInput,
				fmt.Sprintf("Failed %s on %s len", e.Field(), e.Tag()))
			return
		}
	}

	var shopID int
	query := `select id from shops where user_id=$1`
	err = ph.db.QueryRow(r.Context(), query, userID).Scan(&shopID)
	if err != nil {
		log.Println(err.Error())
		reuse.Error(w, http.StatusNotFound, reuse.ErrNotFound, "shop not found please create first")
		return
	}

	var productID int
	query2 := `insert into products (shop_id,name,title,price,mrp,description)
	values ($1,$2,$3,$4,$5,$6) returning id`
	err = ph.db.QueryRow(r.Context(), query2, shopID, input.Name, input.Title, input.Price, input.Mrp, input.Description).Scan(&productID)
	if err != nil {
		log.Println(err.Error())
		reuse.Error(w, http.StatusInternalServerError, reuse.ErrDBFailure, "Product add failed")
		return
	}

	reuse.Success(w, "Product Added Successfully", map[string]any{
		"productID": productID,
	})
}
