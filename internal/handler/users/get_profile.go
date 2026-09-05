package users

import (
	"net/http"
	"shopMe/internal/middleware"
	"shopMe/internal/reuse"
)

type profileOutput struct {
	UserID int
	Name   string
	Email  string
}

func (uh *UserHandler) GetProfile(w http.ResponseWriter, r *http.Request) {

	userId := r.Context().Value(middleware.UserIDContextKey).(int)

	var output profileOutput
	qurey := `select id,name,email from users where id=$1`
	err := uh.db.QueryRow(r.Context(), qurey, userId).Scan(&output.UserID, &output.Name, &output.Email)
	if err != nil {
		reuse.Error(w, http.StatusNotFound, reuse.ErrNotFound, "User not found")
		return
	}

	reuse.Success(w, "Profile get successfully", map[string]any{
		"data": output,
	})
}
