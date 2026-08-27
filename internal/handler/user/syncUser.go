package user

import (
	"context"
	"encoding/json"
	"net/http"
	"shopMe/internal/middleware"
	"shopMe/internal/models"
)

type SyncUserRequest struct {
	Name  string `json:"name"`
	Email string `json:"email"`
}

func (uh UserHandler) SyncUser(w http.ResponseWriter, r *http.Request) {
	firebaseUID, ok := r.Context().Value(middleware.FirebaseUIDKey).(string)
	if !ok {
		http.Error(w, "Unauthorized: firebaseUID missing", http.StatusUnauthorized)
		return
	}

	var req SyncUserRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	var user models.User
	query := `SELECT id, firebase_uid, name, email FROM users WHERE firebase_uid = $1`
	err := uh.db.QueryRow(context.Background(), query, firebaseUID).Scan(&user.ID, &user.FirebaseUID, &user.Name, &user.Email)

	if err != nil {
		insertQuery := `INSERT INTO users (firebase_uid, name, email, created_at, updated_at)
						VALUES ($1, $2, $3, NOW(), NOW())
						RETURNING id, firebase_uid, name, email`
		err = uh.db.QueryRow(context.Background(), insertQuery, firebaseUID, req.Name, req.Email).Scan(
			&user.ID, &user.FirebaseUID, &user.Name, &user.Email,
		)
		if err != nil {
			http.Error(w, "Database error while syncing user: "+err.Error(), http.StatusInternalServerError)
			return
		}
	} else {
		updateQuery := `UPDATE users SET name = $1, email = $2, updated_at = NOW() WHERE firebase_uid = $3`
		_, _ = uh.db.Exec(context.Background(), updateQuery, req.Name, req.Email, firebaseUID)
		user.Name = req.Name
		user.Email = req.Email
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"success": true,
		"user":    user,
	})
}
