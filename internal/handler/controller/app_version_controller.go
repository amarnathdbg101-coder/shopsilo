// Package controller handles HTTP endpoints for application updates and APK distribution.
package controller

import (
	"encoding/json"
	"net/http"
	"os"
	"path/filepath"
	"strconv"
	"strings"

	"shopMe/internal/handler/model"
	"shopMe/internal/handler/services"
	"shopMe/internal/reuse"

	"github.com/go-chi/chi/v5"
	"go.uber.org/zap"
)

// AppVersionController handles version checking and APK artifact streaming.
type AppVersionController struct {
	service      *services.AppVersionService
	downloadsDir string
	logger       *zap.Logger
}

// NewAppVersionController initializes a new AppVersionController.
func NewAppVersionController(service *services.AppVersionService, downloadsDir string, logger *zap.Logger) *AppVersionController {
	// Ensure downloads directory exists on disk
	if downloadsDir == "" {
		downloadsDir = "./downloads"
	}
	_ = os.MkdirAll(downloadsDir, 0755)

	return &AppVersionController{
		service:      service,
		downloadsDir: downloadsDir,
		logger:       logger,
	}
}

// CheckVersion handles GET /api/v1/app/version-check
// It receives client parameters: app, platform, version_code.
func (c *AppVersionController) CheckVersion(w http.ResponseWriter, r *http.Request) {
	q := r.URL.Query()
	appType := q.Get("app")
	if appType == "" {
		appType = "merchant_pos"
	}
	platform := q.Get("platform")
	if platform == "" {
		platform = "android"
	}

	versionCodeStr := q.Get("version_code")
	versionCode, _ := strconv.Atoi(versionCodeStr)

	resp, err := c.service.CheckUpdate(r.Context(), appType, platform, versionCode)
	if err != nil {
		c.logger.Error("version check handler error", zap.Error(err))
		reuse.Error(w, http.StatusInternalServerError, "failed to check app version")
		return
	}

	reuse.Success(w, "version check completed", resp)
}

// ServeDownloadAPK handles GET /downloads/{filename}
// It streams requested APK files using Go's standard http.ServeFile, supporting HTTP Range requests.
func (c *AppVersionController) ServeDownloadAPK(w http.ResponseWriter, r *http.Request) {
	filename := chi.URLParam(r, "filename")
	if filename == "" {
		filename = filepath.Base(r.URL.Path)
	}

	// Sanitize against directory traversal attacks (e.g. ../../etc/passwd)
	cleanFilename := filepath.Base(filepath.Clean(filename))
	if cleanFilename == "." || cleanFilename == "/" || strings.Contains(cleanFilename, "..") {
		reuse.Error(w, http.StatusBadRequest, "invalid filename")
		return
	}

	fullPath := filepath.Join(c.downloadsDir, cleanFilename)

	fileInfo, err := os.Stat(fullPath)
	if err != nil {
		if os.IsNotExist(err) {
			c.logger.Warn("requested APK file not found", zap.String("path", fullPath))
			reuse.Error(w, http.StatusNotFound, "requested APK update file not found")
			return
		}
		reuse.Error(w, http.StatusInternalServerError, "error reading update file")
		return
	}

	if fileInfo.IsDir() {
		reuse.Error(w, http.StatusBadRequest, "invalid file request")
		return
	}

	// Android Package Archive MIME type
	w.Header().Set("Content-Type", "application/vnd.android.package-archive")
	w.Header().Set("Content-Disposition", "attachment; filename=\""+cleanFilename+"\"")
	w.Header().Set("Cache-Control", "public, max-age=3600")

	// http.ServeFile automatically manages byte ranges, resumable downloads, and stream chunks
	http.ServeFile(w, r, fullPath)
}

// PublishRelease handles POST /api/v1/admin/app/releases to register a new APK release.
func (c *AppVersionController) PublishRelease(w http.ResponseWriter, r *http.Request) {
	var input model.AppVersion
	if err := json.NewDecoder(r.Body).Decode(&input); err != nil {
		reuse.Error(w, http.StatusBadRequest, "invalid request body")
		return
	}

	res, err := c.service.PublishRelease(r.Context(), &input)
	if err != nil {
		reuse.Error(w, http.StatusBadRequest, err.Error())
		return
	}

	reuse.Created(w, "app version published successfully", res)
}
