// Package controller handler http work.
package controller

import (
	"errors"
	"mime/multipart"
	"net/http"
	"shopMe/internal/handler/services"
	"shopMe/internal/middleware"
	"shopMe/internal/reuse"
)

type UploadController struct {
	service *services.UploadService
}

func NewUploadController(service *services.UploadService) *UploadController {
	return &UploadController{service: service}
}

// UploadUserAvatar handles uploading user profile image (1 image only)
func (c *UploadController) UploadUserAvatar(w http.ResponseWriter, r *http.Request) {
	claims := middleware.GetUserFromContext(r.Context())
	if claims == nil || claims.UserID == "" {
		reuse.Error(w, http.StatusUnauthorized, "unauthorized")
		return
	}

	// Max 5MB parsing limit
	if err := r.ParseMultipartForm(5 * 1024 * 1024); err != nil {
		reuse.Error(w, http.StatusBadRequest, "failed to parse multipart form")
		return
	}

	file, header, err := r.FormFile("avatar")
	if err != nil {
		reuse.Error(w, http.StatusBadRequest, "avatar file is required")
		return
	}
	defer file.Close()

	avatarURL, err := c.service.UploadUserAvatar(r.Context(), claims.UserID, file, header)
	if err != nil {
		if errors.Is(err, reuse.ErrImageTooLarge) || errors.Is(err, reuse.ErrInvalidImageType) {
			reuse.Error(w, http.StatusBadRequest, err.Error())
			return
		}
		reuse.Error(w, http.StatusInternalServerError, err.Error())
		return
	}

	reuse.Success(w, "Avatar uploaded successfully", map[string]string{
		"avatar_url": avatarURL,
	})
}

// UploadShopImages handles uploading 1 Logo and up to 2 Promotional Banners
func (c *UploadController) UploadShopImages(w http.ResponseWriter, r *http.Request) {
	claims := middleware.GetUserFromContext(r.Context())
	if claims == nil || claims.UserID == "" {
		reuse.Error(w, http.StatusUnauthorized, "unauthorized")
		return
	}

	// Max 10MB parsing limit
	if err := r.ParseMultipartForm(10 * 1024 * 1024); err != nil {
		reuse.Error(w, http.StatusBadRequest, "failed to parse multipart form")
		return
	}

	var logoFile multipart.File
	var logoHeader *multipart.FileHeader

	lf, lh, err := r.FormFile("logo")
	if err == nil {
		logoFile = lf
		logoHeader = lh
		defer logoFile.Close()
	}

	var bannerFiles []multipart.File
	var bannerHeaders []*multipart.FileHeader

	bHeaders := r.MultipartForm.File["banners"]
	if len(bHeaders) > 2 {
		reuse.Error(w, http.StatusBadRequest, "cannot upload more than 2 promotional banners")
		return
	}

	for _, bh := range bHeaders {
		bf, err := bh.Open()
		if err != nil {
			reuse.Error(w, http.StatusBadRequest, "failed to read banner file")
			return
		}
		defer bf.Close()
		bannerFiles = append(bannerFiles, bf)
		bannerHeaders = append(bannerHeaders, bh)
	}

	if logoFile == nil && len(bannerFiles) == 0 {
		reuse.Error(w, http.StatusBadRequest, "at least a logo or banners must be provided")
		return
	}

	logoURL, banners, err := c.service.UploadShopImages(
		r.Context(),
		claims.UserID,
		logoFile,
		logoHeader,
		bannerFiles,
		bannerHeaders,
	)
	if err != nil {
		if errors.Is(err, reuse.ErrImageTooLarge) || errors.Is(err, reuse.ErrInvalidImageType) {
			reuse.Error(w, http.StatusBadRequest, err.Error())
			return
		}
		reuse.Error(w, http.StatusInternalServerError, err.Error())
		return
	}

	reuse.Success(w, "Shop images uploaded successfully", map[string]interface{}{
		"logo_url": logoURL,
		"banners":  banners,
	})
}

// UploadProductImages handles uploading up to 4 images for a product
func (c *UploadController) UploadProductImages(w http.ResponseWriter, r *http.Request) {
	claims := middleware.GetUserFromContext(r.Context())
	if claims == nil || claims.UserID == "" {
		reuse.Error(w, http.StatusUnauthorized, "unauthorized")
		return
	}

	if err := r.ParseMultipartForm(10 * 1024 * 1024); err != nil {
		reuse.Error(w, http.StatusBadRequest, "failed to parse multipart form")
		return
	}

	headers := r.MultipartForm.File["images"]
	if len(headers) == 0 {
		reuse.Error(w, http.StatusBadRequest, "at least one image is required")
		return
	}
	if len(headers) > 4 {
		reuse.Error(w, http.StatusBadRequest, "cannot upload more than 4 images for a product")
		return
	}

	var files []multipart.File
	for _, h := range headers {
		f, err := h.Open()
		if err != nil {
			reuse.Error(w, http.StatusBadRequest, "failed to read product image")
			return
		}
		defer f.Close()
		files = append(files, f)
	}

	urls, err := c.service.UploadProductImages(r.Context(), claims.UserID, files, headers)
	if err != nil {
		if errors.Is(err, reuse.ErrImageTooLarge) || errors.Is(err, reuse.ErrInvalidImageType) {
			reuse.Error(w, http.StatusBadRequest, err.Error())
			return
		}
		reuse.Error(w, http.StatusInternalServerError, err.Error())
		return
	}

	reuse.Success(w, "Product images uploaded successfully", map[string]interface{}{
		"images": urls,
	})
}
