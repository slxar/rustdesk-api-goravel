package api

import (
	"net/http"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/slxar/rustdesk-api-goravel/v3/model"
	"github.com/slxar/rustdesk-api-goravel/v3/service"
)

func MintWebAccess(c *gin.Context) {
	c.Header("Cache-Control", "no-store")
	if !strings.HasPrefix(c.GetHeader("Authorization"), "Bearer ") {
		c.AbortWithStatus(http.StatusUnauthorized)
		return
	}
	var input struct {
		ID        string `json:"id" binding:"required"`
		ExpiresIn int    `json:"expires_in"`
	}
	c.Request.Body = http.MaxBytesReader(c.Writer, c.Request.Body, 1024)
	if c.ShouldBindJSON(&input) != nil {
		c.JSON(400, gin.H{"error": "invalid request"})
		return
	}
	if input.ExpiresIn == 0 {
		input.ExpiresIn = 300
	}
	if input.ExpiresIn < 1 || input.ExpiresIn > 300 {
		c.JSON(400, gin.H{"error": "expires_in must be 1 to 300 seconds"})
		return
	}
	var integrationToken *model.IntegrationToken
	if value, exists := c.Get("integrationToken"); exists {
		integrationToken, _ = value.(*model.IntegrationToken)
	}
	link, err := service.IssueWebLinkForIntegration(service.AllService.UserService.CurUser(c), input.ID, time.Duration(input.ExpiresIn)*time.Second, integrationToken)
	if err != nil {
		c.JSON(403, gin.H{"error": "target access denied or web client is not configured"})
		return
	}
	c.JSON(201, gin.H{"url": link, "id": input.ID, "expires_in": input.ExpiresIn, "single_use": true, "session_expires_in": 3600})
}
