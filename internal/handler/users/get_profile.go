package users

import (
	"net/http"
	"shopMe/internal/middleware"
	"shopMe/internal/reuse"
)

type profileOutput struct {
	UserID   int    `json:"UserID"`
	Name     string `json:"Name"`
	Email    string `json:"Email"`
	ImageURL string `json:"image_url"`
}

func (uh *UserHandler) GetProfile(w http.ResponseWriter, r *http.Request) {
	userId, ok := r.Context().Value(middleware.UserIDContextKey).(int)
	if !ok {
		reuse.Error(w, http.StatusUnauthorized, reuse.ErrUnauthorized, "Unauthorized")
		return
	}

	var output profileOutput
	query := `
		SELECT u.id, u.name, u.email, COALESCE(ui.image_url, '')
		FROM users u
		LEFT JOIN user_images ui ON u.id = ui.user_id
		WHERE u.id = $1
	`
	err := uh.db.QueryRow(r.Context(), query, userId).Scan(
		&output.UserID, &output.Name, &output.Email, &output.ImageURL,
	)
	if err != nil {
		reuse.Error(w, http.StatusNotFound, reuse.ErrNotFound, "User not found")
		return
	}

	reuse.Success(w, "Profile get successfully", map[string]any{
		"data": output,
	})
}
