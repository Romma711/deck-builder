package user

import (
	"github.com/Romma711/deck-builder/pkg/middleware"
	"github.com/gin-gonic/gin"
)

func RegisterRoutes(router *gin.RouterGroup) {
	// Public routes
	router.POST("/register", HandleRegisterUser)
	router.POST("/login", HandleLoginUser)

	// Protected routes that require authentication
	// Create a group for protected routes
	// These routes will use the JWT middleware to verify the token
	protected := router.Group("/")
	protected.Use(middleware.JwtAuthMiddleware())

	// Protected routes 
	protected.PUT("/users/:id", HandleUpdateUserByID)
	protected.DELETE("/users/:id", HandleDeleteUserByID)
}