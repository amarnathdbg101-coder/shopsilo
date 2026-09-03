package users

import (
	"encoding/json"
	"net/http"
	"shopMe/internal/middleware"
	"shopMe/internal/reuseable"

	"golang.org/x/crypto/bcrypt"
)

type password struct {
	NewPassword string `json:"new_password"`
	OldPassword string `json:"old_password"`
}

func (uh *UserHandler) ChangePassword(w http.ResponseWriter, r *http.Request) {
	userId, ok := r.Context().Value(middleware.UserIDContextKey).(int)
	if !ok || userId == 0 {
		reuseable.Error(w,http.StatusUnauthorized,"unauthorized","Unauthorized user")
		return
	}

	// parse input
	var input password
	if err := json.NewDecoder(r.Body).Decode(&input); err != nil {
		reuseable.Error(w,http.StatusBadRequest,"invalid input","Invalid data")
		return
	}

	if input.NewPassword == input.OldPassword {
		reuseable.Error(w,http.StatusBadRequest,"both password same please try another one","Invalid data")
        return
	}

	// fetch old password hash
	var oldPassword string
	query1 := `SELECT password FROM users WHERE id=$1`
	if err := uh.db.QueryRow(r.Context(), query1, userId).Scan(&oldPassword); err != nil {
		reuseable.Error(w, http.StatusNotFound, "User not found", "not found")
		return
	}

	// verify old password
	if err := bcrypt.CompareHashAndPassword([]byte(oldPassword), []byte(input.OldPassword)); err != nil {
		reuseable.Error(w, http.StatusBadRequest, "Old password wrong", "Invalid_error")
		return
	}

	// hash new password
	newPassword, err := reuseable.HashPassword(input.NewPassword)
	if err != nil {
		reuseable.Error(w, http.StatusInternalServerError, "Password hashing failed", "internal_error")
		return
	}

	// update DB
	query2 := `UPDATE users SET password=$1 WHERE id=$2`
	if _, err := uh.db.Exec(r.Context(), query2, newPassword, userId); err != nil {
		reuseable.Error(w, http.StatusInternalServerError, "Password update failed", "internal_error")
		return
	}

	// response
	_ = json.NewEncoder(w).Encode(map[string]any{
		"status":  "success",
		"message": "password updated successfully",
	})
}
