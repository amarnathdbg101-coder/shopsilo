package controller

import (
	"encoding/base64"
	"encoding/json"
	"io"
	"net/http"
	"strings"

	"shopMe/internal/handler/dto"
	"shopMe/internal/handler/services"
	"shopMe/internal/reuse"
)

type AIController struct {
	aiService *services.AIService
}

func NewAIController(aiService *services.AIService) *AIController {
	return &AIController{aiService: aiService}
}

// ScanProduct processes product packaging photos to auto-fill title, brand, category, MRP, and unit.
func (c *AIController) ScanProduct(w http.ResponseWriter, r *http.Request) {
	var req dto.AIScanProductRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		reuse.Error(w, http.StatusBadRequest, "invalid request body")
		return
	}

	if req.ImageBase64 == "" {
		reuse.Error(w, http.StatusBadRequest, "image_base64 is required")
		return
	}

	resp, err := c.aiService.ScanProduct(r.Context(), req)
	if err != nil {
		reuse.Error(w, http.StatusInternalServerError, err.Error())
		return
	}

	reuse.Success(w, "Product packet scanned successfully", resp)
}

// SmartStudioScan handles rapid camera capture or file upload for studio background clean-up and spec extraction.
func (c *AIController) SmartStudioScan(w http.ResponseWriter, r *http.Request) {
	var req dto.SmartStudioScanRequest

	// Check if multipart form was sent (direct camera photo upload)
	if strings.HasPrefix(r.Header.Get("Content-Type"), "multipart/form-data") {
		// Parse max 10MB
		if err := r.ParseMultipartForm(10 * 1024 * 1024); err == nil {
			file, header, fileErr := r.FormFile("image")
			if fileErr == nil {
				defer file.Close()
				imgBytes, _ := io.ReadAll(file)
				req.ImageBase64 = base64.StdEncoding.EncodeToString(imgBytes)
				req.MimeType = header.Header.Get("Content-Type")
				req.ThemeHint = r.FormValue("theme_hint")
			}
		}
	} else {
		// JSON payload with Base64 image
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			reuse.Error(w, http.StatusBadRequest, "invalid request body")
			return
		}
	}

	if req.ImageBase64 == "" {
		reuse.Error(w, http.StatusBadRequest, "image is required (as multipart form 'image' or json 'image_base64')")
		return
	}

	resp, err := c.aiService.SmartStudioScan(r.Context(), req)
	if err != nil {
		reuse.Error(w, http.StatusInternalServerError, err.Error())
		return
	}

	reuse.Success(w, "Product smart studio scan completed successfully", resp)
}

// ScanMandiItem processes snapshots of empty shelves, low stock items, or cartons to generate wholesale procurement proposals.
func (c *AIController) ScanMandiItem(w http.ResponseWriter, r *http.Request) {
	var req dto.AIMandiScanRequest

	if strings.HasPrefix(r.Header.Get("Content-Type"), "multipart/form-data") {
		if err := r.ParseMultipartForm(10 * 1024 * 1024); err == nil {
			file, header, fileErr := r.FormFile("image")
			if fileErr == nil {
				defer file.Close()
				imgBytes, _ := io.ReadAll(file)
				req.ImageBase64 = base64.StdEncoding.EncodeToString(imgBytes)
				req.MimeType = header.Header.Get("Content-Type")
			}
		}
	} else {
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			reuse.Error(w, http.StatusBadRequest, "invalid request body")
			return
		}
	}

	if req.ImageBase64 == "" {
		reuse.Error(w, http.StatusBadRequest, "image is required (as multipart form 'image' or json 'image_base64')")
		return
	}

	resp, err := c.aiService.ScanMandiItem(r.Context(), req)
	if err != nil {
		reuse.Error(w, http.StatusInternalServerError, err.Error())
		return
	}

	reuse.Success(w, "Mandi sourcing items extracted successfully", resp)
}
