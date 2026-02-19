package middleware

import (
	"net/http"

	"github.com/gin-gonic/gin"
)

func RequireRoles(roles ...string) gin.HandlerFunc {
	allow := make(map[string]struct{}, len(roles))
	for _, role := range roles {
		allow[role] = struct{}{}
	}

	return func(c *gin.Context) {
		roleAny, exists := c.Get("role")
		if !exists {
			c.AbortWithStatusJSON(http.StatusForbidden, gin.H{"error": "permission denied"})
			return
		}
		role, _ := roleAny.(string)
		if _, ok := allow[role]; !ok {
			c.AbortWithStatusJSON(http.StatusForbidden, gin.H{"error": "permission denied"})
			return
		}
		c.Next()
	}
}
