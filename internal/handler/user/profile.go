package user

import (
	"context"
	"encoding/json"
	"net/http"
	"shopMe/internal/middleware"
	"shopMe/internal/models"
)

func (uh UserHandler) Profile(w http.ResponseWriter, r *http.Request) {
	userId, ok := r.Context().Value(middleware.UserIDKey).(int)
	if !ok {
		http.Error(w, "Unauthorized: userId missing in context", http.StatusUnauthorized)
		return
	}

	var user models.User
	query := `
		SELECT u.id, u.firebase_uid, u.name, u.email, s.id, s.name, s.shop_custom_id
		FROM users u
		LEFT JOIN shops s ON u.id = s.user_id
		WHERE u.id = $1`

	var shopID *int
	var shopName, shopCustomID *string

	err := uh.db.QueryRow(context.Background(), query, userId).Scan(
		&user.ID, &user.FirebaseUID, &user.Name, &user.Email,
		&shopID, &shopName, &shopCustomID,
	)

	if err != nil {
		http.Error(w, "User not found or database error", http.StatusInternalServerError)
		return
	}

	if shopID != nil {
		user.Shop = &models.Shop{
			ID:           *shopID,
			UserID:       user.ID,
			Name:         *shopName,
			ShopCustomID: *shopCustomID,
		}
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"success": true,
		"user":    user,
	})
}
