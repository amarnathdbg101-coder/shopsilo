// Package model contains core database entity structs.
package model

import "time"

// AppVersion represents a published version release of ShopSilo applications.
type AppVersion struct {
	ID                      int64     `json:"id"`
	AppType                 string    `json:"app_type"`                   // e.g. "merchant_pos", "customer_app"
	Platform                string    `json:"platform"`                   // e.g. "android", "ios", "windows"
	VersionCode             int       `json:"version_code"`               // Integer build-number (e.g. 101)
	VersionName             string    `json:"version_name"`               // Semantic version string (e.g. "1.0.1")
	MinSupportedVersionCode int       `json:"min_supported_version_code"` // Minimum version code required to operate
	ApkURL                  string    `json:"apk_url"`                    // Download URL or relative static path
	FileSizeBytes           int64     `json:"file_size_bytes"`            // Total file size in bytes
	ReleaseNotes            []string  `json:"release_notes"`              // List of changelog bullets in Hinglish/English
	IsForceUpdate           bool      `json:"is_force_update"`            // True if update is strictly mandatory
	IsActive                bool      `json:"is_active"`                  // Soft-delete or enable/disable flag
	CreatedAt               time.Time `json:"created_at"`
}

// VersionCheckResponse is the JSON payload delivered to Flutter apps during startup update checks.
type VersionCheckResponse struct {
	UpdateAvailable         bool     `json:"update_available"`
	LatestVersion           string   `json:"latest_version"`
	LatestCode              int      `json:"latest_code"`
	CurrentCode             int      `json:"current_code"`
	IsForceUpdate           bool     `json:"is_force_update"`
	DownloadURL             string   `json:"download_url"`
	FileSizeBytes           int64    `json:"file_size_bytes"`
	ReleaseNotes            []string `json:"release_notes"`
	MinSupportedVersionCode int      `json:"min_supported_version_code"`
}
