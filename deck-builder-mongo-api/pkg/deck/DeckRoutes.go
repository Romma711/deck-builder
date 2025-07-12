package deck

import (
	"github.com/Romma711/deck-builder/pkg/middleware"
	"github.com/gin-gonic/gin"
)

func RegisterRoutes(router *gin.RouterGroup) {
	// Public routes
	router.GET("/decks/:id", HandleGetDeckByID)
	router.GET("/decks/owner/:owner_id", HandleGetDecksByOwnerID)
	router.GET("/decks/format", HandleGetDecksByFormat)
	router.GET("/decks/commander", HandleGetDecksByCommander)
	router.GET("/decks", HandleGetAllDecks)
	// Create a group for protected routes
	// These routes require authentication
	// and will use the JWT middleware to verify the token
	protected := router.Group("/")
	protected.Use(middleware.JwtAuthMiddleware())

	// Protected routes
	protected.PUT("/decks/:id", HandleUpdateDeck)
	protected.POST("/decks", HandleCreateDeck)
	protected.DELETE("/decks/:id", HandleDeleteDeck)
}
