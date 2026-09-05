package shop

import (
	"net/http"
	"shopMe/internal/middleware"
	"shopMe/internal/reuse"
	"strconv"

	"github.com/go-chi/chi/v5"
)

type ShopResponse struct {
	ID        int      `json:"id"`
	UserID    int      `json:"user_id"`
	Name      string   `json:"name"`
	Timing    string   `json:"timing"`
	Category  string   `json:"category"`
	Village   string   `json:"village"`
	Post      string   `json:"post"`
	District  string   `json:"district"`
	State     string   `json:"state"`
	Pincode   string   `json:"pincode"`
	Landmark  string   `json:"landmark"`
	Latitude  float64  `json:"latitude"`
	Longitude float64  `json:"longitude"`
	Images    []string `json:"images"`
}

func (sh *ShopHandler) GetMyShop(w http.ResponseWriter, r *http.Request) {
	userID, ok := r.Context().Value(middleware.UserIDContextKey).(int)
	if !ok {
		reuse.Error(w, http.StatusUnauthorized, reuse.ErrUnauthorized, "Unauthorized")
		return
	}

	query := `
		SELECT s.id, s.user_id, s.name, COALESCE(s.timing, ''), COALESCE(s.category, ''),
		       a.village, COALESCE(a.post, ''), a.district, a.state, COALESCE(a.pincode, ''),
		       COALESCE(a.landmark, ''), COALESCE(a.latitude, 0), COALESCE(a.longitude, 0)
		FROM shops s
		JOIN addresses a ON s.address_id = a.id
		WHERE s.user_id = $1
	`

	var res ShopResponse
	err := sh.db.QueryRow(r.Context(), query, userID).Scan(
		&res.ID, &res.UserID, &res.Name, &res.Timing, &res.Category,
		&res.Village, &res.Post, &res.District, &res.State, &res.Pincode,
		&res.Landmark, &res.Latitude, &res.Longitude,
	)
	if err != nil {
		reuse.Error(w, http.StatusNotFound, reuse.ErrNotFound, "Shop not found")
		return
	}

	// Fetch shop images
	imgQuery := `SELECT image_url FROM shop_images WHERE shop_id = $1`
	rows, err := sh.db.Query(r.Context(), imgQuery, res.ID)
	res.Images = []string{}
	if err == nil {
		defer rows.Close()
		for rows.Next() {
			var imgURL string
			if err := rows.Scan(&imgURL); err == nil {
				res.Images = append(res.Images, imgURL)
			}
		}
	}

	reuse.Success(w, "Shop fetched successfully", res)
}

func (sh *ShopHandler) GetShopByID(w http.ResponseWriter, r *http.Request) {
	shopIDStr := chi.URLParam(r, "id")
	shopID, err := strconv.Atoi(shopIDStr)
	if err != nil {
		reuse.Error(w, http.StatusBadRequest, reuse.ErrInvalidInput, "Invalid shop ID")
		return
	}

	query := `
		SELECT s.id, s.user_id, s.name, COALESCE(s.timing, ''), COALESCE(s.category, ''),
		       a.village, COALESCE(a.post, ''), a.district, a.state, COALESCE(a.pincode, ''),
		       COALESCE(a.landmark, ''), COALESCE(a.latitude, 0), COALESCE(a.longitude, 0)
		FROM shops s
		JOIN addresses a ON s.address_id = a.id
		WHERE s.id = $1
	`

	var res ShopResponse
	err = sh.db.QueryRow(r.Context(), query, shopID).Scan(
		&res.ID, &res.UserID, &res.Name, &res.Timing, &res.Category,
		&res.Village, &res.Post, &res.District, &res.State, &res.Pincode,
		&res.Landmark, &res.Latitude, &res.Longitude,
	)
	if err != nil {
		reuse.Error(w, http.StatusNotFound, reuse.ErrNotFound, "Shop not found")
		return
	}

	// Fetch shop images
	imgQuery := `SELECT image_url FROM shop_images WHERE shop_id = $1`
	rows, err := sh.db.Query(r.Context(), imgQuery, res.ID)
	res.Images = []string{}
	if err == nil {
		defer rows.Close()
		for rows.Next() {
			var imgURL string
			if err := rows.Scan(&imgURL); err == nil {
				res.Images = append(res.Images, imgURL)
			}
		}
	}

	reuse.Success(w, "Shop fetched successfully", res)
}
