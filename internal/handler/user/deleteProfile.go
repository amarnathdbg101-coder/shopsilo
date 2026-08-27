package user

import (
	"context"
	"net/http"
	"shopMe/internal/middleware"
)

func (uh UserHandler) DeleteAccount(w http.ResponseWriter, r *http.Request) {
	userId, ok := r.Context().Value(middleware.UserIDKey).(int)
	if !ok {
		http.Error(w, "Unauthorized: userId missing in context", http.StatusUnauthorized)
		return
	}

	query := `DELETE FROM users WHERE id = $1`
	result, err := uh.db.Exec(context.Background(), query, userId)
	if err != nil {
		http.Error(w, "data delete nhi hua", http.StatusInternalServerError)
		return
	}

	if result.RowsAffected() == 0 {
		http.Error(w, "no user deleted", http.StatusNotFound)
		return
	}

	w.WriteHeader(http.StatusOK)
	_, _ = w.Write([]byte("User deleted successfully"))
}
