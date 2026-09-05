package products

import (
	"net/http"
	"shopMe/internal/reuse"
	"strconv"

	"github.com/go-chi/chi/v5"
)

func (ph *ProductHandler) GetProduct(w http.ResponseWriter, r *http.Request) {
	productIDStr := chi.URLParam(r, "id")
	productID, err := strconv.Atoi(productIDStr)
	if err != nil {
		reuse.Error(w, http.StatusBadRequest, reuse.ErrInvalidInput, "Invalid product ID")
		return
	}

	query := `SELECT id, shop_id, name, COALESCE(title, ''), price, mrp, COALESCE(category, ''), COALESCE(description, '') FROM products WHERE id = $1`

	var p ProductResponse
	err = ph.db.QueryRow(r.Context(), query, productID).Scan(&p.ID, &p.ShopID, &p.Name, &p.Title, &p.Price, &p.Mrp, &p.Category, &p.Description)
	if err != nil {
		reuse.Error(w, http.StatusNotFound, reuse.ErrNotFound, "Product not found")
		return
	}

	p.Images = []string{}
	imgRows, errImg := ph.db.Query(r.Context(), "SELECT image_url FROM product_images WHERE product_id = $1", p.ID)
	if errImg == nil {
		defer imgRows.Close()
		for imgRows.Next() {
			var url string
			if err := imgRows.Scan(&url); err == nil {
				p.Images = append(p.Images, url)
			}
		}
	}

	reuse.Success(w, "Product fetched successfully", p)
}
