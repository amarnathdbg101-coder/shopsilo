package products

import (
	"encoding/json"
	"net/http"
	"shopMe/internal/middleware"
	"shopMe/internal/reuse"
	"strconv"

	"github.com/go-chi/chi/v5"
)

type productUpdateInput struct {
	Name        string  `json:"name"`
	Title       string  `json:"title"`
	Price       float64 `json:"price"`
	Mrp         float64 `json:"mrp"`
	Category    string  `json:"category"`
	Description string  `json:"description"`
}

func (ph *ProductHandler) UpdateProduct(w http.ResponseWriter, r *http.Request) {
	userID, ok := r.Context().Value(middleware.UserIDContextKey).(int)
	if !ok {
		reuse.Error(w, http.StatusUnauthorized, reuse.ErrUnauthorized, "Unauthorized")
		return
	}

	productIDStr := chi.URLParam(r, "id")
	productID, err := strconv.Atoi(productIDStr)
	if err != nil {
		reuse.Error(w, http.StatusBadRequest, reuse.ErrInvalidInput, "Invalid product ID")
		return
	}

	// Verify user owns the shop that owns this product
	var shopID int
	checkQuery := `SELECT p.shop_id FROM products p JOIN shops s ON p.shop_id = s.id WHERE p.id = $1 AND s.user_id = $2`
	err = ph.db.QueryRow(r.Context(), checkQuery, productID, userID).Scan(&shopID)
	if err != nil {
		reuse.Error(w, http.StatusForbidden, reuse.ErrForbidden, "You do not own this product")
		return
	}

	var input productUpdateInput
	if err := json.NewDecoder(r.Body).Decode(&input); err != nil {
		reuse.Error(w, http.StatusBadRequest, reuse.ErrInvalidInput, "Invalid JSON input")
		return
	}

	updateQuery := `UPDATE products SET name=$1, title=$2, price=$3, mrp=$4, category=$5, description=$6 WHERE id=$7`
	_, err = ph.db.Exec(r.Context(), updateQuery, input.Name, input.Title, input.Price, input.Mrp, input.Category, input.Description, productID)
	if err != nil {
		reuse.Error(w, http.StatusInternalServerError, reuse.ErrDBFailure, "Failed to update product: "+err.Error())
		return
	}

	reuse.Success(w, "Product updated successfully", map[string]any{"product_id": productID})
}
