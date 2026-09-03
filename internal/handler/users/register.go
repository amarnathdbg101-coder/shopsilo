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
		reuseable.Error(w,http.StatusBadRequest,"Something went wrong","Invalid_data")
		return
	}

	hashedPassword ,err := reuseable.HashPassword(input.Password)
	if err != nil {
		reuseable.Error(w,http.StatusBadRequest,"Password hashing failed","internal_error")
		return
	}

	// Insert register input into database
	query := `INSERT INTO users (name, email, password) VALUES ($1, $2, $3)`
	_,err =uh.db.Exec(r.Context(),query,input.Name,input.Email,hashedPassword)
	if err != nil {
		reuseable.Error(w,http.StatusInternalServerError,"Email already exists","internal_error")
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

	_ = json.NewEncoder(w).Encode(res)

}