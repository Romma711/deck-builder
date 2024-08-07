package routes

import(
	"github.com/gorilla/mux"
	"deck-builder-back/pkg/controller"
)

func userRoutes(router mux.Router, user controller.Handler){
	router.HandleFunc("/login",user.HandleLogin)


}