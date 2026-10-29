package api

import (
	"github.com/gin-gonic/gin"
	"github.com/looplj/axonhub/internal/ent"
	"github.com/looplj/axonhub/internal/objects"
	"net/http"
	"time"
)

func (h *SystemHandlers) UserUsageDetails(c *gin.Context) {
	guid, err := objects.ParseGUID(c.Query("userId"))
	if err != nil || guid.Type != ent.TypeUser || guid.ID <= 0 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid userId"})
		return
	}
	var from, to *time.Time
	for _, item := range []struct {
		name string
		dest **time.Time
	}{{"from", &from}, {"to", &to}} {
		if value := c.Query(item.name); value != "" {
			t, err := time.Parse(time.RFC3339Nano, value)
			if err != nil {
				c.JSON(http.StatusBadRequest, gin.H{"error": "invalid date"})
				return
			}
			*item.dest = &t
		}
	}
	data, err := h.SystemService.UserUsageDetails(c.Request.Context(), guid.ID, from, to)
	if err != nil {
		c.JSON(http.StatusForbidden, gin.H{"error": "unable to retrieve usage details"})
		return
	}
	c.JSON(http.StatusOK, data)
}
