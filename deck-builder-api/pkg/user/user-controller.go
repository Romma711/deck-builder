package user

import (
	"deck-builder-back/pkg/controller/middlewere"
	"deck-builder-back/pkg/types"
	"encoding/json"
	"net/http"
	"strconv"

	"github.com/gorilla/mux"
)

type Handler struct {
	store types.UserStore
}

func NewHandler(store types.UserStore) *Handler {
	return &Handler{store: store}
}

func (h *Handler) HandleLogin(w http.ResponseWriter, r *http.Request) {
	var user types.ToLogin
	json.NewDecoder(r.Body).Decode(&user)
	res,_ := h.store.GetUserByEmail(user.Email)
	if res == nil {
		w.Header().Add("Content-Type", "application/json")
		w.WriteHeader(http.StatusUnauthorized)
		json.NewEncoder(w).Encode("Email o Password son invalidos1")
		return
	}
	if !middleware.ComparePasswords(res.Password, []byte(user.Password)) {
		w.Header().Add("Content-Type", "application/json")
		w.WriteHeader(http.StatusUnauthorized)
		json.NewEncoder(w).Encode("Email o Password son invalidos2")
		return
	}
	w.Header().Add("Content-Type", "application/json")
	w.WriteHeader(http.StatusAccepted)
	json.NewEncoder(w).Encode("Has iniciado sesion con exito")
}

func (h *Handler) HandleRegister(w http.ResponseWriter, r *http.Request) {
	///validar que el mail sea valido
	var user types.ToRegister
	json.NewDecoder(r.Body).Decode(&user)
	_, err := h.store.GetUserByEmail(user.Email)
	if err == nil {
		w.Header().Add("Content-Type", "application/json")
		w.WriteHeader(http.StatusExpectationFailed)
		json.NewEncoder(w).Encode("El usuario ya se encuentra registrado.")
		return
	}
	hashedPass, err := middleware.HashPassword(user.Password)
	if err != nil{
		w.Header().Add("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode("Error al codificar")
		return
		///devolver un error del servidor
	}

	err = h.store.CreateUser(types.User{
		Alias:    user.Alias,
		Email:    user.Email,
		Password: hashedPass,
	})
	if err != nil {
		w.Header().Add("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(err)
		return
		///devolver un error del servidor
	}
	w.Header().Add("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode("La cuenta se creo exitosamente")
}

func (h *Handler) HandleGetUser(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)

	id, ok := vars["id"]
	if !ok {
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode("Falta el ID a buscar")
		return
	}
	userID, err := strconv.Atoi(id)
	if !ok {
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode("Falta el ID a buscar")
		return
	}
	user, err := h.store.GetUserById(userID)
	if err != nil {
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode("No se encontro al cliente")
		return
	}
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(user)

}
