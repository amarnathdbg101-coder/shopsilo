package controller

import (
	"net/http"
	"strings"

	"shopMe/internal/handler/services"
	"shopMe/internal/reuse"

	"github.com/go-chi/chi/v5"
)

type CatalogController struct {
	service *services.CatalogService
}

func NewCatalogController(service *services.CatalogService) *CatalogController {
	return &CatalogController{service: service}
}

// Lookup handles GET /catalog/barcode/{code} or GET /catalog/lookup?barcode={code}
func (c *CatalogController) Lookup(w http.ResponseWriter, r *http.Request) {
	code := chi.URLParam(r, "code")
	if strings.TrimSpace(code) == "" {
		code = r.URL.Query().Get("barcode")
	}
	clean := strings.TrimSpace(code)
	if clean == "" {
		reuse.Error(w, http.StatusBadRequest, "barcode parameter is required")
		return
	}

	res, err := c.service.Lookup(r.Context(), clean)
	if err != nil {
		reuse.Error(w, http.StatusNotFound, "product not found in master catalog")
		return
	}

	reuse.Success(w, "Master catalog product resolved", res)
}

// Search handles GET /catalog/search?q=...
func (c *CatalogController) Search(w http.ResponseWriter, r *http.Request) {
	query := r.URL.Query().Get("q")
	if query == "" {
		query = r.URL.Query().Get("search")
	}
	clean := strings.TrimSpace(query)
	if clean == "" {
		reuse.Error(w, http.StatusBadRequest, "search query parameter 'q' is required")
		return
	}

	res, err := c.service.Search(r.Context(), clean, 20)
	if err != nil {
		reuse.Error(w, http.StatusInternalServerError, err.Error())
		return
	}

	reuse.Success(w, "Master catalog search results retrieved", res)
}
