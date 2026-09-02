package users

import (
	"encoding/json"
	"net/http"
	"shopMe/internal/middleware"
	"shopMe/internal/reuseable"

	"golang.org/x/crypto/bcrypt"
)
type password struct {
    NewPassword string `json:"new_password"`
    OldPassword string `json:"old_password"`
}

func (uh *UserHandler) ChangePassword(w http.ResponseWriter, r *http.Request) {
    userId, ok := r.Context().Value(middleware.UserIDContextKey).(int)
    if !ok || userId == 0 {
        http.Error(w, `{"status":"error","message":"unauthorized"}`, http.StatusUnauthorized)
        return
    }

    // parse input
    var input password
    if err := json.NewDecoder(r.Body).Decode(&input); err != nil {
        http.Error(w, `{"status":"error","message":"Invalid data"}`, http.StatusBadRequest)
        return
    }

    if input.NewPassword == input.OldPassword {
        http.Error(w, `{"status":"error","message":"old and new password both same, please enter another new password"}`, http.StatusBadRequest)
        return
    }

    // fetch old password hash
    var oldPassword string
    query1 := `SELECT password FROM users WHERE id=$1`
    if err := uh.db.QueryRow(r.Context(), query1, userId).Scan(&oldPassword); err != nil {
        http.Error(w, `{"status":"error","message":"user not found"}`, http.StatusBadRequest)
        return
    }

    // verify old password
    if err := bcrypt.CompareHashAndPassword([]byte(oldPassword), []byte(input.OldPassword)); err != nil {
        http.Error(w, `{"status":"error","message":"wrong old password"}`, http.StatusBadRequest)
        return
    }

    // hash new password
    newPassword, err := reuseable.HashPassword(input.NewPassword)
    if err != nil {
        http.Error(w, `{"status":"error","message":"password hashing failed"}`, http.StatusInternalServerError)
        return
    }

    // update DB
    query2 := `UPDATE users SET password=$1 WHERE id=$2`
    if _, err := uh.db.Exec(r.Context(), query2, newPassword, userId); err != nil {
        http.Error(w, `{"status":"error","message":"password update failed"}`, http.StatusInternalServerError)
        return
    }

    // response
    if err := json.NewEncoder(w).Encode(map[string]any{
        "status":  "success",
        "message": "password updated successfully",
    }); err != nil {
        http.Error(w, "response error", http.StatusInternalServerError)
        return
    }
}
