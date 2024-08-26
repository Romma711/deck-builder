package main

import (
	"database/sql"
	"deck-builder-back/pkg/config"
	"deck-builder-back/pkg/user"
	"deck-builder-back/pkg/deck"
	"fmt"
	"log"
	"net/http"

	"github.com/go-sql-driver/mysql"
	"github.com/gorilla/mux"
)

func main(){
	env:=config.LoadConfig()
	
	cfg := mysql.Config{
		User: env.DBUser,
		Passwd: env.DBPassword,
		Addr: fmt.Sprintf("%s:%s",env.DBHost,env.Port),
		DBName: env.DBName,
		AllowNativePasswords: true,
	}

	db, err := NewMySQLStorage(cfg)
	if err != nil{
		log.Fatal()
	}

	initStorage(db)

	userModel:=user.NewStore(db)
	userController:= user.NewHandler(userModel)

	deckModel:=deck.NewStore(db)
	deckController:=deck.NewHandler(deckModel)

	r := mux.NewRouter()
	subr:= r.PathPrefix("/api").Subrouter()

	subr.HandleFunc("/login",userController.HandleLogin).Methods("POST")
	subr.HandleFunc("/register",userController.HandleRegister).Methods("POST")
	subr.HandleFunc("/user/{id}",userController.HandleGetUser).Methods("GET")
	subr.HandleFunc("/deck/{name}", deckController.HandleGetDeckByName).Methods("GET")
	subr.HandleFunc("/deck/{id}", deckController.HandleGetDeckByUser).Methods("GET")
	subr.HandleFunc("/deck/create", deckController.HandleCreateNewDeck).Methods("POST")
	log.Fatal(http.ListenAndServe("localhost:9010", r))
}

func NewMySQLStorage(cfg mysql.Config)(*sql.DB, error){
	db, err:= sql.Open("mysql",cfg.FormatDSN())
	if err != nil{
		log.Fatal(err)
	}

	return db, nil
}

func initStorage (db *sql.DB){
	err:= db.Ping()
	if err != nil{
		log.Fatal(err)
	}
	log.Println("DB: Conectado correctamente!")
}