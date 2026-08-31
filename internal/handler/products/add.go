package products

import (
	"net/http"
)

func (ph *ProductHandler) AddProduct(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	_, err := w.Write([]byte("Product added successfully"))
	if err != nil {
		http.Error(w, "response error", http.StatusBadRequest)
		return
	}
}
