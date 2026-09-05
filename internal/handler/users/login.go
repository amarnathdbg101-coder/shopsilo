package users

import (
	"encoding/json"
	"fmt"
	"net/http"
	"shopMe/internal/logger"
	"shopMe/internal/reuse"

	"github.com/go-playground/validator/v10"
	"golang.org/x/crypto/bcrypt"
)

type loginData struct {
	UserID   int
	Password string
}
type loginInput struct {
	Email    string
	Password string
}

func (uh *UserHandler) Login(w http.ResponseWriter, r *http.Request) {
	// parse input
	var input loginInput
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
	// fetch data from database
	var data loginData
	query := `select id,password from users where email=$1`
	err = uh.db.QueryRow(r.Context(), query, input.Email).Scan(&data.UserID, &data.Password)
	if err != nil {
		reuse.Error(w, http.StatusBadRequest, reuse.ErrInvalidInput, "Wrong password")
		return
	}
	// check password
	err = bcrypt.CompareHashAndPassword([]byte(data.Password), []byte(input.Password))
	if err != nil {
		reuse.Error(w, http.StatusBadRequest, reuse.ErrInvalidInput, "Wrong Password")
		return
	}

	// generate jwt Token
	token, _ := reuse.GenerateJwt(data.UserID, input.Email)

	reuse.Success(w, "Welcome In ShopSilo", map[string]any{
		"token": token,
	})
}
