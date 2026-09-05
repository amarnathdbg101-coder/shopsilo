package products

import (
	"net/http"
	"shopMe/internal/reuse"
	"strconv"
)

type ProductResponse struct {
	ID          int      `json:"id"`
	ShopID      int      `json:"shop_id"`
	Name        string   `json:"name"`
	Title       string   `json:"title"`
	Price       float64  `json:"price"`
	Mrp         float64  `json:"mrp"`
	Category    string   `json:"category"`
	Description string   `json:"description"`
	Images      []string `json:"images"`
}

func (ph *ProductHandler) GetProducts(w http.ResponseWriter, r *http.Request) {
	shopIDStr := r.URL.Query().Get("shop_id")
	category := r.URL.Query().Get("category")

	shopID := 0
	if shopIDStr != "" {
		if id, err := strconv.Atoi(shopIDStr); err == nil {
			shopID = id
		}
	}

	query := `
		SELECT p.id, p.shop_id, p.name, COALESCE(p.title, ''), p.price, p.mrp,
		       COALESCE(p.category, ''), COALESCE(p.description, ''),
		       COALESCE(ARRAY_AGG(pi.image_url) FILTER (WHERE pi.image_url IS NOT NULL), '{}') AS images
		FROM products p
		LEFT JOIN product_images pi ON p.id = pi.product_id
		WHERE ($1::int = 0 OR p.shop_id = $1)
		  AND ($2::text = '' OR p.category = $2)
		GROUP BY p.id
		ORDER BY p.id DESC
	`

	rows, err := ph.db.Query(r.Context(), query, shopID, category)
	if err != nil {
		reuse.Error(w, http.StatusInternalServerError, reuse.ErrDBFailure, "Failed to fetch products: "+err.Error())
		return
	}
	defer rows.Close()

	products := []ProductResponse{}

	for rows.Next() {
		var p ProductResponse
		var imgArray []string
		err := rows.Scan(
			&p.ID, &p.ShopID, &p.Name, &p.Title, &p.Price, &p.Mrp,
			&p.Category, &p.Description, &imgArray,
		)
		if err == nil {
			p.Images = imgArray
			products = append(products, p)
		}
	}

	reuse.Success(w, "Products fetched successfully", products)
}
