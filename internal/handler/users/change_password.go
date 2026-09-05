package users

import (
	"encoding/json"
	"fmt"
	"net/http"
	"shopMe/internal/middleware"
	"shopMe/internal/reuse"

	"github.com/go-playground/validator/v10"
	"golang.org/x/crypto/bcrypt"
)

type password struct {
	NewPassword string `json:"new_password" validate:"required,min=8"`
	OldPassword string `json:"old_password" validate:"required,min=8"`
}

func (uh *UserHandler) ChangePassword(w http.ResponseWriter, r *http.Request) {
	userId, ok := r.Context().Value(middleware.UserIDContextKey).(int)
	if !ok || userId == 0 {
		reuse.Error(w, http.StatusUnauthorized, reuse.ErrUnauthorized, "unauthorized")
		return
	}

	// parse input
	var input password
	if err := json.NewDecoder(r.Body).Decode(&input); err != nil {
		reuse.Error(w, http.StatusBadRequest, reuse.ErrInvalidInput, "invalid input")
		return
	}
	// Check input
	err := reuse.Validate.Struct(input)
	if err != nil {
		for _, e := range err.(validator.ValidationErrors) {
			reuse.Error(w, http.StatusBadRequest, reuse.ErrInvalidInput,
				fmt.Sprintf("Failed %s on %s len", e.Field(), e.Tag()))
			return
		}

	}

	if input.NewPassword == input.OldPassword {
		reuse.Error(w, http.StatusBadRequest, reuse.ErrInvalidInput, "both password same please try another one")
		return
	}

	// fetch old password hash
	var oldPassword string
	query1 := `SELECT password FROM users WHERE id=$1`
	if err := uh.db.QueryRow(r.Context(), query1, userId).Scan(&oldPassword); err != nil {
		reuse.Error(w, http.StatusNotFound, reuse.ErrNotFound, "User not found")
		return
	}

	// verify old password
	if err := bcrypt.CompareHashAndPassword([]byte(oldPassword), []byte(input.OldPassword)); err != nil {
		reuse.Error(w, http.StatusBadRequest, reuse.ErrInvalidInput, "Old password wrong")
		return
	}

	// hash new password
	newPassword, err := reuse.HashPassword(input.NewPassword)
	if err != nil {
		reuse.Error(w, http.StatusInternalServerError, reuse.ErrInternal, "Password hashing failed")
		return
	}

	// update DB
	query2 := `UPDATE users SET password=$1 WHERE id=$2`
	if _, err := uh.db.Exec(r.Context(), query2, newPassword, userId); err != nil {
		reuse.Error(w, http.StatusInternalServerError, reuse.ErrDBFailure, "Password update failed")
		return
	}

	// response
	reuse.Success(w, "Password change successfully", map[string]any{
		"userID": userId,
	})
}
