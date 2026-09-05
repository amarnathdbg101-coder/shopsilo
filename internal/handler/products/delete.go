package products

import (
	"net/http"
	"shopMe/internal/middleware"
	"shopMe/internal/reuse"
	"strconv"

	"github.com/go-chi/chi/v5"
)

func (ph *ProductHandler) DeleteProduct(w http.ResponseWriter, r *http.Request) {
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

	deleteQuery := `DELETE FROM products WHERE id = $1`
	_, err = ph.db.Exec(r.Context(), deleteQuery, productID)
	if err != nil {
		reuse.Error(w, http.StatusInternalServerError, reuse.ErrDBFailure, "Failed to delete product: "+err.Error())
		return
	}

	reuse.Success(w, "Product deleted successfully", map[string]any{"product_id": productID})
}
