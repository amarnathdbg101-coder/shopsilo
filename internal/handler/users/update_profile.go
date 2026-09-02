package users

import (
	"encoding/json"
	"net/http"
	"shopMe/internal/middleware"
)

type profileUpdate struct {
    Name  string `json:"name"`
    Email string `json:"email"`
}

func (uh *UserHandler) UpdateProfile(w http.ResponseWriter, r *http.Request) {
    userID, ok := r.Context().Value(middleware.UserIDContextKey).(int)
    if !ok || userID == 0 {
        http.Error(w, `{"status":"error","message":"unauthorized"}`, http.StatusUnauthorized)
        return
    }

    var input profileUpdate
    if err := json.NewDecoder(r.Body).Decode(&input); err != nil {
        http.Error(w, `{"status":"error","message":"invalid input"}`, http.StatusBadRequest)
        return
    }

    // Update DB
    query := `UPDATE users SET name=$1, email=$2 WHERE id=$3`
    _, err := uh.db.Exec(r.Context(), query, input.Name, input.Email, userID)
    if err != nil {
        http.Error(w, `{"status":"error","message":"update failed"}`, http.StatusInternalServerError)
        return
    }

    // Response
    if err := json.NewEncoder(w).Encode(map[string]any{
        "status":  "success",
        "message": "profile updated successfully",
        "data":    input,
    }); err != nil {
        http.Error(w, "response error", http.StatusInternalServerError)
        return
    }
}
