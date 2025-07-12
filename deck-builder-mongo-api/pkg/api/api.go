package api

import (
	"log"

	"github.com/Romma711/deck-builder/pkg/deck"
	"github.com/Romma711/deck-builder/pkg/user"
	"github.com/gin-gonic/gin"
)

func StartServer(){
	
	r := gin.Default()

	subr := r.Group("/api/v2")

	user.RegisterRoutes(subr)
	deck.RegisterRoutes(subr)

	log.Fatal(r.Run(":8080")) // Run on port 8080
	log.Println("Servidor iniciado en el puerto 8080")
}