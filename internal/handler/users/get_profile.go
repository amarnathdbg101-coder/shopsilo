package users

import (
	"encoding/json"
	"net/http"
	"shopMe/internal/middleware"
	"shopMe/internal/reuseable"
)

func (uh *UserHandler) GetProfile(w http.ResponseWriter, r *http.Request){
	
	userId := r.Context().Value(middleware.UserIDContextKey).(int)

	var output ProfileOutput
	qurey := `select id,name,email from users where id=$1`
	err := uh.db.QueryRow(r.Context(),qurey,userId).Scan(&output.UserId,&output.Name,&output.Email)
	if err != nil {
		reuseable.Error(w,http.StatusNotFound,"User not found","Data not found")
		return
	}

	_ = json.NewEncoder(w).Encode(map[string]any{
		"status":"success",
		"message":"Profile fetch successfully",
		"data": output,
	})
}