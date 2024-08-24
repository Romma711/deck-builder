package deck

import (
	"deck-builder-back/pkg/types"
	"encoding/json"
	"net/http"
	"strconv"

	"github.com/gorilla/mux"
)

type Handler struct {
	store types.DeckStore
}

func NewHandler(store types.DeckStore) *Handler {
	return &Handler{store: store}
}

func (h *Handler) handleCreateNewDeck(w http.ResponseWriter, r *http.Request) {
	var deck types.Deck
	json.NewDecoder(r.Body).Decode(&deck)
	switch deck.Format{
	case 1:
		if deck.DeckSize != 100{
			w.Header().Add("Content-Type", "application/json")
			w.WriteHeader(http.StatusUnauthorized)
			json.NewEncoder(w).Encode("Cantidad de cartas en el mazo invalidas")
			return
		}
	case 2, 3, 4, 5:
		if deck.DeckSize < 60{
			w.Header().Add("Content-Type", "application/json")
			w.WriteHeader(http.StatusUnauthorized)
			json.NewEncoder(w).Encode("Cantidad de cartas en el mazo invalidas")
			return
		}
	}
	err := h.store.CreateDeck(types.Deck{
		Name: deck.Name,
		Format: deck.Format,
		DeckSize: deck.DeckSize,
		CreatedBy: deck.CreatedBy,
		DeckList: deck.DeckList,
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
	json.NewEncoder(w).Encode("Deck succesfully created")
}
