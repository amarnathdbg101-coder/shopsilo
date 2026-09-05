package users

import (
	"encoding/json"
	"fmt"
	"net/http"
	"shopMe/internal/middleware"
	"shopMe/internal/reuse"

	"github.com/go-playground/validator/v10"
)

type profileUpdate struct {
	Name  string `json:"name"`
	Email string `json:"email"`
}

func (uh *UserHandler) UpdateProfile(w http.ResponseWriter, r *http.Request) {
	userID, ok := r.Context().Value(middleware.UserIDContextKey).(int)
	if !ok {
		reuse.Error(w, http.StatusUnauthorized, "unauthorized", "unauthorized_error")
		return
	}

	var input profileUpdate
	err := json.NewDecoder(r.Body).Decode(&input)
	if err != nil {
		reuse.Error(w, http.StatusBadRequest, "Invalid input", "Invalid_data")
		return
	}
	// check input
	err = reuse.Validate.Struct(input)
	if err != nil {
		for _, e := range err.(validator.ValidationErrors) {
			reuse.Error(w, http.StatusBadRequest, reuse.ErrInvalidInput,
				fmt.Sprintf("Failed %s on %s len", e.Field(), e.Tag()))
			return
		}

	}
	// Update DB
	query := `UPDATE users SET name=$1, email=$2 WHERE id=$3`
	_, err = uh.db.Exec(r.Context(), query, input.Name, input.Email, userID)
	if err != nil {
		reuse.Error(w, http.StatusInternalServerError, "profile update failed", "internal_error")
		return
	}

	// Response
	reuse.Success(w, "Profile Update successfully", map[string]any{
		"userID": userID,
	})
}
