package user

import (
	"log"
	"net/http"

	"github.com/Romma711/deck-builder/pkg/types"
	"github.com/gin-gonic/gin"
)

func HandleRegisterUser(c *gin.Context) {
	var registerPayload types.RegisterPayload
	if err := c.ShouldBindJSON(&registerPayload); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid input"})
		return
	}

	user, err := RegisterUser(registerPayload)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusCreated, user)
}

func HandleLoginUser(c *gin.Context) {
	var loginPayload types.LoginPayload
	if err := c.ShouldBindJSON(&loginPayload); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid input"})
		return
	}

	token, err := LoginUser(loginPayload)
	if err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, map[string]string{"token": "Bearer " + token})
}

func HandleUpdateUserByID(c *gin.Context) {
	var updatePayload types.UpdateUserPayload
	if err := c.ShouldBindJSON(&updatePayload); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid input"})
		return
	}

	userID := c.Param("id")
	updatedUser, err := UpdateUser(userID, updatePayload)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, updatedUser)
}

func HandleDeleteUserByID(c *gin.Context) {
	userID := c.Param("id")
	log.Printf("Deleting user with ID: %s", userID)
	err := DeleteUser(userID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusNoContent, nil)
}