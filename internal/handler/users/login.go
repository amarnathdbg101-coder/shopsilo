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
		http.Error(w,`{"status":"error","message":"Invalid data"}`,http.StatusBadRequest)
		return
	}
	// fetch data from database
	var data RegisterInput
	query := `select id,password from users where email=$1`
	err = uh.db.QueryRow(r.Context(),query,input.Email).Scan(&data.UserId,&data.Password)
	if err != nil {
		http.Error(w,`{"status":"error","message":"Wrong email ,please enter right email"}`,http.StatusInternalServerError)
		return
	}
	// check password
	err = bcrypt.CompareHashAndPassword([]byte(data.Password),[]byte(input.Password))
	if err != nil {
		http.Error(w,`{"status":"error","message":"wrong Password"}`,http.StatusBadRequest)
		return
	}

	// generate jwt Token
	token ,_ := reuseable.GenerateJwt(data.UserId,input.Email)


	err = json.NewEncoder(w).Encode(&token)
	if err != nil {
		http.Error(w,`{"status":"error","message":"response error"}`,http.StatusInternalServerError)
		return
	}
}