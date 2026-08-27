package handler

import (
	"context"
	"encoding/json"
	"net/http"
	"shopMe/internal/middleware"
)

func (uh UserHandler) Profile(w http.ResponseWriter, r *http.Request) {
	userId, ok := r.Context().Value(middleware.UserIDKey).(int)
	if !ok {
		http.Error(w, "Unauthorized: userId missing in context", http.StatusUnauthorized)
		return
	}
	var user RegisterOutput
	query := `select id,name,email from users where id=$1`
	err := uh.db.QueryRow(context.Background(), query, userId).Scan(&user.Id, &user.Name, &user.Email)
	if err != nil {
		http.Error(w, "database throw error", http.StatusInternalServerError)
		return
	}

	w.Header().Set("content-type","application/json")
	err = json.NewEncoder(w).Encode(&user)
	if err != nil{
		http.Error(w,err.Error(),http.StatusBadRequest)
	}


}