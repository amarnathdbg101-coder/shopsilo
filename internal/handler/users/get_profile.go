package users

import (
	"encoding/json"
	"net/http"
	"shopMe/internal/middleware"
)

func (uh *UserHandler) GetProfile(w http.ResponseWriter, r *http.Request){
	
	userId := r.Context().Value(middleware.UserIDContextKey).(int)

	var output ProfileOutput
	qurey := `select id,name,email from users where id=$1`
	err := uh.db.QueryRow(r.Context(),qurey,userId).Scan(&output.UserId,&output.Name,&output.Email)
	if err != nil {
		http.Error(w,`{"status":"error","message":"user not found"}`,http.StatusInternalServerError)
		return
	}

	err = json.NewEncoder(w).Encode(map[string]any{
		"status":"success",
		"message":"Profile fetch successfully",
		"data": output,
	})
	if err != nil {
		http.Error(w,"response error",http.StatusInternalServerError)
		return
	}
}