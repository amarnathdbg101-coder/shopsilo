package shop

import (
	"encoding/json"
	"net/http"
	"shopMe/internal/middleware"
	"shopMe/internal/reuse"
)

type Input struct {
	ShopName string `json:"name"`
	Timing   string `json:"timing"`
	Category string `json:"category"`
}

func (sh *ShopHandler) UpdateShop(w http.ResponseWriter, r *http.Request) {
	userId, ok := r.Context().Value(middleware.UserIDContextKey).(int)
	if !ok {
		reuse.Error(w, http.StatusUnauthorized, reuse.ErrUnauthorized, "Unauthorized")
		return
	}

	var input Input
	if err := json.NewDecoder(r.Body).Decode(&input); err != nil {
		reuse.Error(w, http.StatusBadRequest, reuse.ErrInvalidInput, "Invalid JSON body")
		return
	}

	query := `
        UPDATE shops 
        SET name=$1, timing=$2, category=$3, updated_at=NOW()
        WHERE user_id=$4
        RETURNING id
    `

	var shopId int
	err := sh.db.QueryRow(r.Context(), query,
		input.ShopName, input.Timing, input.Category, userId,
	).Scan(&shopId)

	if err != nil {
		reuse.Error(w, http.StatusInternalServerError, reuse.ErrDBFailure, "Update failed: "+err.Error())
		return
	}

	reuse.Success(w, "Shop updated successfully", map[string]any{"shopId": shopId})
}
