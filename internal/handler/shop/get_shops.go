package shop

import (
	"net/http"
	"shopMe/internal/reuse"
)

func (sh *ShopHandler) GetShops(w http.ResponseWriter, r *http.Request) {
	category := r.URL.Query().Get("category")

	query := `
		SELECT s.id, s.user_id, s.name, COALESCE(s.timing, ''), COALESCE(s.category, ''),
		       a.village, COALESCE(a.post, ''), a.district, a.state, COALESCE(a.pincode, ''),
		       COALESCE(a.landmark, ''), COALESCE(a.latitude, 0), COALESCE(a.longitude, 0)
		FROM shops s
		JOIN addresses a ON s.address_id = a.id
	`
	var args []interface{}

	if category != "" {
		query += " WHERE s.category = $1"
		args = append(args, category)
	}

	query += " ORDER BY s.id DESC"

	rows, err := sh.db.Query(r.Context(), query, args...)
	if err != nil {
		reuse.Error(w, http.StatusInternalServerError, reuse.ErrDBFailure, "Failed to fetch shops")
		return
	}
	defer rows.Close()

	var shops []ShopResponse = []ShopResponse{}

	for rows.Next() {
		var res ShopResponse
		err := rows.Scan(
			&res.ID, &res.UserID, &res.Name, &res.Timing, &res.Category,
			&res.Village, &res.Post, &res.District, &res.State, &res.Pincode,
			&res.Landmark, &res.Latitude, &res.Longitude,
		)
		if err == nil {
			// Fetch images for each shop
			imgQuery := `SELECT image_url FROM shop_images WHERE shop_id = $1`
			imgRows, errImg := sh.db.Query(r.Context(), imgQuery, res.ID)
			res.Images = []string{}
			if errImg == nil {
				for imgRows.Next() {
					var imgURL string
					if err := imgRows.Scan(&imgURL); err == nil {
						res.Images = append(res.Images, imgURL)
					}
				}
				imgRows.Close()
			}
			shops = append(shops, res)
		}
	}

	reuse.Success(w, "Shops fetched successfully", shops)
}
