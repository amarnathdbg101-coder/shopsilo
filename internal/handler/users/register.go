package users

import (
	"encoding/json"
	"fmt"
	"net/http"
	"shopMe/internal/logger"
	"shopMe/internal/reuse"

	"github.com/go-playground/validator/v10"
)

type registerInput struct {
	UserID   int `json:"user_id"`
	Name     string `json:"name"`
	Email    string `json:"email" validate:"required,email"`
	Password string `json:"password" validate:"required,min=8"`
}

func (uh *UserHandler) Register(w http.ResponseWriter, r *http.Request) {
	// input parse
	var input registerInput
	err := json.NewDecoder(r.Body).Decode(&input)
	if err != nil {
		logger.New().Error("register input")
		reuse.Error(w, http.StatusBadRequest, reuse.ErrInvalidInput, "Invalid input")
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

	hashedPassword, err := reuse.HashPassword(input.Password)
	if err != nil {
		reuse.Error(w, http.StatusBadRequest, reuse.ErrInternal, "Password hashing failed")
		return
	}

	// Insert register input into database
	query := `INSERT INTO users (name, email, password) VALUES ($1, $2, $3)`
	_, err = uh.db.Exec(r.Context(), query, input.Name, input.Email, hashedPassword)
	if err != nil {
		reuse.Error(w, http.StatusBadRequest, reuse.ErrDBFailure, "Email already exists")
		return
	}

	reuse.Success(w, "Register successfully", map[string]any{
		"name":  input.Name,
		"email": input.Email,
	})

}
