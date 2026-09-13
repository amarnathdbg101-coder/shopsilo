package controller

import (
	"net/http"
	"shopMe/internal/middleware"
	"shopMe/internal/reuse"
	"shopMe/internal/utils"

	"github.com/jackc/pgx/v5/pgxpool"
	"go.uber.org/zap"
)

type BackupController struct {
	db     *pgxpool.Pool
	logger *zap.Logger
}

func NewBackupController(db *pgxpool.Pool, logger *zap.Logger) *BackupController {
	return &BackupController{
		db:     db,
		logger: logger,
	}
}

// TriggerOnDemandBackup generates a full database backup dump and uploads to Cloudflare R2 (Protected - Admin)
func (c *BackupController) TriggerOnDemandBackup(w http.ResponseWriter, r *http.Request) {
	claims := middleware.GetUserFromContext(r.Context())
	if claims == nil || claims.Role != "admin" {
		reuse.Error(w, http.StatusUnauthorized, "admin access required")
		return
	}

	meta, err := utils.GenerateFullDatabaseBackupDump(r.Context(), c.db)
	if err != nil {
		c.logger.Error("failed to generate on-demand database backup", zap.Error(err))
		reuse.Error(w, http.StatusInternalServerError, err.Error())
		return
	}

	reuse.Success(w, "Cloudflare R2 database backup generated successfully", meta)
}

// ListCloudBackups returns all available automated cloud backups (Protected - Admin)
func (c *BackupController) ListCloudBackups(w http.ResponseWriter, r *http.Request) {
	claims := middleware.GetUserFromContext(r.Context())
	if claims == nil || claims.Role != "admin" {
		reuse.Error(w, http.StatusUnauthorized, "admin access required")
		return
	}

	list, err := utils.ListCloudR2Backups(r.Context())
	if err != nil {
		c.logger.Error("failed to list cloud R2 backups", zap.Error(err))
		reuse.Error(w, http.StatusInternalServerError, err.Error())
		return
	}

	reuse.Success(w, "Cloud database backups listed successfully", list)
}
