package api

import (
	"net/http"

	"nofx/maintenance"

	"github.com/gin-gonic/gin"
)

type updateMaintenanceCleanupRequest struct {
	Enabled *bool `json:"enabled"`
}

func (s *Server) handleGetMaintenanceCleanup(c *gin.Context) {
	cfg, err := maintenance.ReadAutoCleanupConfig(s.store)
	if err != nil {
		SafeInternalError(c, "Failed to load cleanup config", err)
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"enabled":   cfg.Enabled,
		"days":      cfg.Days,
		"supported": cfg.Supported,
	})
}

func (s *Server) handleUpdateMaintenanceCleanup(c *gin.Context) {
	var req updateMaintenanceCleanupRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request body"})
		return
	}
	if req.Enabled == nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "enabled is required"})
		return
	}

	if err := maintenance.SetAutoCleanupEnabled(s.store, *req.Enabled); err != nil {
		SafeInternalError(c, "Failed to update cleanup config", err)
		return
	}

	cfg, err := maintenance.ReadAutoCleanupConfig(s.store)
	if err != nil {
		SafeInternalError(c, "Failed to load cleanup config", err)
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"enabled":   cfg.Enabled,
		"days":      cfg.Days,
		"supported": cfg.Supported,
	})
}

