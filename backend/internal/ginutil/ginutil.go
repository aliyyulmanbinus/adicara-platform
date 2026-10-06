package ginutil

import (
	"errors"
	"log"
	"net/http"

	"github.com/gin-gonic/gin"

	"github.com/aliyyulmanbinus/adicara-platform/backend/internal/apierr"
)

func Error(c *gin.Context, err error) {
	var ae *apierr.Error
	if errors.As(err, &ae) {
		c.JSON(ae.Status, gin.H{"error": ae})
		return
	}
	log.Printf("request failed: %v", err)
	c.JSON(http.StatusInternalServerError, gin.H{"error": apierr.New(500, "internal", "internal server error")})
}
