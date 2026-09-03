package users

import (
	"encoding/json"
	"net/http"
	"shopMe/internal/logger"
	"shopMe/internal/reuseable"

	"golang.org/x/crypto/bcrypt"
)

func (uh *UserHandler) Login(w http.ResponseWriter, r *http.Request){
	// parse input
	var input LoginInput
	err := json.NewDecoder(r.Body).Decode(&input)
		if err != nil {
		logger.New().Error("register input")
		reuseable.Error(w,http.StatusBadRequest,"Invalid input","Invalid data")
		return
	}
	// fetch data from database
	var data RegisterInput
	query := `select id,password from users where email=$1`
	err = uh.db.QueryRow(r.Context(),query,input.Email).Scan(&data.UserId,&data.Password)
	if err != nil {
		reuseable.Error(w,http.StatusBadRequest,"Wrong password","Invalid data")
		return
	}
	// check password
	err = bcrypt.CompareHashAndPassword([]byte(data.Password),[]byte(input.Password))
	if err != nil {
		reuseable.Error(w,http.StatusBadRequest,"Wrong Password","Invalid data")
		return
	}

	// generate jwt Token
	token ,_ := reuseable.GenerateJwt(data.UserId,input.Email)


	_= json.NewEncoder(w).Encode(&token)
}