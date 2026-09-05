package shop

import (
	"net/http"
	"shopMe/internal/middleware"
	"shopMe/internal/reuse"
)

func (sh *ShopHandler) DeleteShop(w http.ResponseWriter, r *http.Request) {
	userID := r.Context().Value(middleware.UserIDContextKey)

	query := `delete from shops where user_id=$1`
	row, err := sh.db.Exec(r.Context(), query, userID)
	if err != nil {
		reuse.Error(w, http.StatusInternalServerError, reuse.ErrNotFound, "User not found")
		return
	}
	if row.RowsAffected() == 0 {
		reuse.Error(w,http.StatusNotFound,reuse.ErrNotFound,"Shop not Found")
		return
	}

	reuse.Success(w, "Shop delete successfully", map[string]any{
		"user_id": userID,
	})
}
