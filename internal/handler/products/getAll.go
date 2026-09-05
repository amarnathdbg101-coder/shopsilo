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

	query := `SELECT id, shop_id, name, COALESCE(title, ''), price, mrp, COALESCE(category, ''), COALESCE(description, '') FROM products WHERE 1=1`
	var args []interface{}
	argCount := 1

	if shopIDStr != "" {
		shopID, err := strconv.Atoi(shopIDStr)
		if err == nil {
			query += ` AND shop_id = $` + strconv.Itoa(argCount)
			args = append(args, shopID)
			argCount++
		}
	}

	if category != "" {
		query += ` AND category = $` + strconv.Itoa(argCount)
		args = append(args, category)
		argCount++
	}

	query += ` ORDER BY id DESC`

	rows, err := ph.db.Query(r.Context(), query, args...)
	if err != nil {
		reuse.Error(w, http.StatusInternalServerError, reuse.ErrDBFailure, "Failed to fetch products: "+err.Error())
		return
	}
	defer rows.Close()

	products := []ProductResponse{}

	for rows.Next() {
		var p ProductResponse
		err := rows.Scan(&p.ID, &p.ShopID, &p.Name, &p.Title, &p.Price, &p.Mrp, &p.Category, &p.Description)
		if err == nil {
			// Fetch images
			p.Images = []string{}
			imgRows, errImg := ph.db.Query(r.Context(), "SELECT image_url FROM product_images WHERE product_id = $1", p.ID)
			if errImg == nil {
				for imgRows.Next() {
					var url string
					if err := imgRows.Scan(&url); err == nil {
						p.Images = append(p.Images, url)
					}
				}
				imgRows.Close()
			}
			products = append(products, p)
		}
	}

	reuse.Success(w, "Products fetched successfully", products)
}
