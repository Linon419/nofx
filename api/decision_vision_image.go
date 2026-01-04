package api

import (
	"database/sql"
	"errors"
	"net/http"
	"os"
	"path/filepath"
	"strconv"
	"strings"

	"nofx/logger"

	"github.com/gin-gonic/gin"
)

func (s *Server) handleDecisionVisionImage(c *gin.Context) {
	_, traderID, err := s.getTraderFromQuery(c)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	trader, err := s.traderManager.GetTrader(traderID)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": err.Error()})
		return
	}

	id, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil || id <= 0 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid decision id"})
		return
	}

	name := strings.TrimSpace(c.Param("name"))
	if name == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "missing image name"})
		return
	}

	record, err := trader.GetStore().Decision().GetRecordByID(trader.GetID(), id)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			c.JSON(http.StatusNotFound, gin.H{"error": "decision not found"})
			return
		}
		logger.Infof("failed to load decision record %d: %v", id, err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to load decision record"})
		return
	}

	var relPath string
	for _, img := range record.VisionImages {
		if img.Name == name {
			relPath = strings.TrimSpace(img.Path)
			break
		}
	}
	if relPath == "" {
		c.JSON(http.StatusNotFound, gin.H{"error": "image not found"})
		return
	}

	cleanRel := filepath.Clean(filepath.FromSlash(relPath))
	if cleanRel == "." || cleanRel == "" || filepath.IsAbs(cleanRel) {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid image path"})
		return
	}

	sep := string(os.PathSeparator)
	if cleanRel != "vision_images" && !strings.HasPrefix(cleanRel, "vision_images"+sep) {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid image path"})
		return
	}

	baseDir, _ := filepath.Abs(filepath.Join("data", "vision_images"))
	fullPath, _ := filepath.Abs(filepath.Join("data", cleanRel))
	if baseDir != "" && fullPath != "" && fullPath != baseDir && !strings.HasPrefix(fullPath, baseDir+sep) {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid image path"})
		return
	}

	if st, err := os.Stat(fullPath); err != nil || st.IsDir() {
		c.JSON(http.StatusNotFound, gin.H{"error": "image not found"})
		return
	}

	c.Header("Cache-Control", "private, max-age=3600")
	c.File(fullPath)
}

