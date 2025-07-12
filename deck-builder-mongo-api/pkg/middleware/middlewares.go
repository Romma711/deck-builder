package middleware

import (
	"log"
	"net/http"
	"strings"

	"github.com/Romma711/deck-builder/pkg/utils"
	"github.com/gin-gonic/gin"
)

func JwtAuthMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		tokenString := c.GetHeader("Authorization")
		if !strings.HasPrefix(tokenString, "Bearer ") {
			log.Println("Authorization header missing or malformed")
			c.String(http.StatusUnauthorized, "Unauthorized")
			c.Abort()
			return
		}

		tokenString = strings.TrimPrefix(tokenString, "Bearer ")
		if tokenString == "" {
			log.Println("Token string is empty")
			c.String(http.StatusUnauthorized, "Unauthorized")
			c.Abort()
			return
		}
		
		err := utils.VerifyToken(tokenString)
		if err != nil {
			log.Printf("JWT verification failed: %v", err)
			c.String(http.StatusUnauthorized, "Unauthorized")
			c.Abort()
			return
		}
		c.Next()
	}
}