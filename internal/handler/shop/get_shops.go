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
		       COALESCE(a.landmark, ''), COALESCE(a.latitude, 0), COALESCE(a.longitude, 0),
		       COALESCE(ARRAY_AGG(si.image_url) FILTER (WHERE si.image_url IS NOT NULL), '{}') AS images
		FROM shops s
		JOIN addresses a ON s.address_id = a.id
		LEFT JOIN shop_images si ON s.id = si.shop_id
		WHERE ($1::text = '' OR s.category = $1)
		GROUP BY s.id, a.id
		ORDER BY s.id DESC
	`

	rows, err := sh.db.Query(r.Context(), query, category)
	if err != nil {
		reuse.Error(w, http.StatusInternalServerError, reuse.ErrDBFailure, "Failed to fetch shops: "+err.Error())
		return
	}
	defer rows.Close()

	shops := []ShopResponse{}

	for rows.Next() {
		var res ShopResponse
		var imgArray []string
		err := rows.Scan(
			&res.ID, &res.UserID, &res.Name, &res.Timing, &res.Category,
			&res.Village, &res.Post, &res.District, &res.State, &res.Pincode,
			&res.Landmark, &res.Latitude, &res.Longitude, &imgArray,
		)
		if err == nil {
			res.Images = imgArray
			shops = append(shops, res)
		}
	}

	reuse.Success(w, "Shops fetched successfully", shops)
}
