package api

import (
	"strings"

	"github.com/gin-gonic/gin"

	"github.com/your-server-support/podman-swarm/internal/security"
)

// AuthMiddleware creates authentication middleware for API
func AuthMiddleware(tokenManager *security.APITokenManager, enabled bool) gin.HandlerFunc {
	return func(c *gin.Context) {
		// Skip authentication if not enabled
		if !enabled {
			c.Next()
			return
		}

		// Skip authentication for health endpoint
		if c.Request.URL.Path == "/api/v1/health" {
			c.Next()
			return
		}

		// Get token from Authorization header
		authHeader := c.GetHeader("Authorization")
		if authHeader == "" {
			c.JSON(401, gin.H{"error": "Missing Authorization header"})
			c.Abort()
			return
		}

		// Extract token (support both "Bearer <token>" and raw token)
		var token string
		if strings.HasPrefix(authHeader, "Bearer ") {
			token = strings.TrimPrefix(authHeader, "Bearer ")
		} else {
			token = authHeader
		}

		// Validate token
		if !tokenManager.ValidateToken(token) {
			c.JSON(401, gin.H{"error": "Invalid or expired token"})
			c.Abort()
			return
		}

		c.Next()
	}
}
