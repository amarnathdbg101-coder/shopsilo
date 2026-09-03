package users

import (
	"encoding/json"
	"net/http"
	"shopMe/internal/middleware"
	"shopMe/internal/reuseable"
)

type profileUpdate struct {
	Name  string `json:"name"`
	Email string `json:"email"`
}

func (uh *UserHandler) UpdateProfile(w http.ResponseWriter, r *http.Request) {
	userID, ok := r.Context().Value(middleware.UserIDContextKey).(int)
	if !ok {
		reuseable.Error(w, http.StatusUnauthorized, "unauthorized", "unauthorized_error")
		return
	}

	var input profileUpdate
	err := json.NewDecoder(r.Body).Decode(&input)
	if err != nil {
		reuseable.Error(w, http.StatusBadRequest, "Invalid input", "Invalid_data")
		return
	}

	// Update DB
	query := `UPDATE users SET name=$1, email=$2 WHERE id=$3`
	_, err = uh.db.Exec(r.Context(), query, input.Name, input.Email, userID)
	if err != nil {
		reuseable.Error(w, http.StatusInternalServerError, "profile update failed", "internal_error")
		return
	}

	// Response
	if err := json.NewEncoder(w).Encode(map[string]any{
		"status":  "success",
		"message": "profile updated successfully",
		"data":    input,
	}); err != nil {
		reuseable.Error(w, http.StatusInternalServerError, "Response error", "internal_error")
		return
	}
}
