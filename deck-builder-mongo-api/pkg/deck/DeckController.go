package deck

import (
	"log"
	"net/http"

	"github.com/Romma711/deck-builder/pkg/types"
	"github.com/Romma711/deck-builder/pkg/utils"
	"github.com/gin-gonic/gin"
)

func HandleCreateDeck(c *gin.Context){
	var deckRequest types.DeckPayload
	if err := c.ShouldBindBodyWithJSON(&deckRequest); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Datos invalidos"})
		return
	}

	authHeader := c.GetHeader("Authorization")
	log.Println("Authorization Header:", authHeader)
	user, err := utils.GetUserFromToken(authHeader)
	if err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Token de autorización inválido " + err.Error()})
		return
	}

	deck, err := CreateDeck(deckRequest, user.ID.Hex())
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Error al crear el mazo: " + err.Error()})
		return
	}

	c.JSON(http.StatusCreated, deck)
}

func HandleGetDeckByID(c *gin.Context) {
	deckID := c.Param("id")
	deck, err := GetDeckByID(deckID)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Mazo no encontrado"})
		return
	}
	c.JSON(http.StatusOK, deck)
}

func HandleGetDecksByCommander(c *gin.Context) {
	commander := c.Query("commander")
	decks, err := GetDecksByCommander(commander)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Mazos no encontrado"})
		return
	}
	c.JSON(http.StatusOK, decks)
}

func HandleGetDecksByOwnerID(c *gin.Context) {
	ownerID := c.Param("owner_id")
	decks, err := GetDecksByOwnerID(ownerID)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "No se encontraron mazos"})
		return
	}
	c.JSON(http.StatusOK, decks)
}

func HandleGetDecksByFormat(c *gin.Context) {
	format := c.Query("format")
	log.Println("Buscando mazos por formato:", format)
	decks, err := GetDecksByFormat(format)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "No se encontraron mazos para este formato"})
		return
	}
	c.JSON(http.StatusOK, decks)
}
/*
func HandleGetAllDecksCommander(c *gin.Context) {
	decks, err := GetAllDecksCommander()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Error al obtener los mazos"})
		return
	}
	c.JSON(http.StatusOK, decks)
}
*/
func HandleGetAllDecks(c *gin.Context) {
	decks, err := GetAllDecks()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Error al obtener los mazos"})
		return
	}
	c.JSON(http.StatusOK, decks)
}

func HandleUpdateDeck(c *gin.Context) {
	var deckRequest types.DeckPayload
	if err := c.ShouldBindBodyWithJSON(&deckRequest); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Datos invalidos"})
		return
	}

	deckID := c.Param("id")
	deck, err := UpdateDeck(deckID, deckRequest)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Error al actualizar el mazo"})
		return
	}

	c.JSON(http.StatusOK, deck)
}

func HandleDeleteDeck(c *gin.Context) {
	deckID := c.Param("id")
	err := DeleteDeck(deckID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Error al eliminar el mazo"})
		return
	}

	c.JSON(http.StatusNoContent, nil)
}