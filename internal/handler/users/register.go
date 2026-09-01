package users

import (
	"encoding/json"
	"net/http"
	"shopMe/internal/logger"
	"shopMe/internal/reuseable"
)

func (uh *UserHandler) Register(w http.ResponseWriter, r *http.Request){
	// input parse
	var input RegisterInput
	err := json.NewDecoder(r.Body).Decode(&input)
	if err != nil {
		logger.New().Error("register input")
		http.Error(w,`{"status":"error","message":"Invalid data"}`,http.StatusBadRequest)
		return
	}

	hashedPassword ,err := reuseable.HashPassword(input.Password)
	if err != nil {
		http.Error(w,`{"status":"error","message":"password hashing error"}`,http.StatusBadRequest)
		return
	}

	// Insert register input into database
	query := `INSERT INTO users (name, email, password) VALUES ($1, $2, $3)`
	_,err =uh.db.Exec(r.Context(),query,input.Name,input.Email,hashedPassword)
	if err != nil {
		http.Error(w,`{"status":"error","message":"Email already exists"}`,http.StatusBadRequest)
		return
	}

	// Response
	res := map[string]any{
		"status":"success",
		"message":"Register successfully",
		"data":map[string]any{
			"name":input.Name,
			"email":input.Email,
		},
	}

	err = json.NewEncoder(w).Encode(res)
	if err != nil {
		http.Error(w,`{"status":"error","message":"response error"}`,http.StatusInternalServerError)
		return
	}

}