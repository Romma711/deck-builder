package deck

import (
	"deck-builder-back/pkg/types"
	"encoding/json"
	"net/http"
)

type Handler struct {
	store types.DeckStore
}

func NewHandler(store types.DeckStore) *Handler {
	return &Handler{store: store}
}

func (h *Handler) HandleCreateNewDeck(w http.ResponseWriter, r *http.Request) {
	var deck types.NewDeck
	json.NewDecoder(r.Body).Decode(&deck)
	switch deck.Format {
	case 1:
		if deck.DeckSize != 100 {
			w.Header().Add("Content-Type", "application/json")
			w.WriteHeader(http.StatusUnauthorized)
			json.NewEncoder(w).Encode("Cantidad de cartas en el mazo invalidas")
			return
		}
	case 2, 3, 4, 5:
		if deck.DeckSize < 60 {
			w.Header().Add("Content-Type", "application/json")
			w.WriteHeader(http.StatusUnauthorized)
			json.NewEncoder(w).Encode("Cantidad de cartas en el mazo invalidas")
			return
		}
	}
	err := h.store.CreateDeck(types.NewDeck{
		Name:      deck.Name,
		Format:    deck.Format,
		DeckSize:  deck.DeckSize,
		CreatedBy: deck.CreatedBy,
		DeckList:  deck.DeckList,
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

func (h *Handler) HandleGetDeckByName(w http.ResponseWriter, r *http.Request) {
	queryParams := r.URL.Query()
	name := queryParams.Get("name")
	res, err := h.store.GetDeckByName(name)
	if err != nil {
		w.Header().Add("Content-Type", "application/json")
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(err)
	}
	w.Header().Add("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(res)
}

func (h *Handler) HandleGetDeckByUser(w http.ResponseWriter, r *http.Request) {
	var id int
	json.NewDecoder(r.Body).Decode(&id)
	res, err := h.store.GetDecksByUser(id)
	if err != nil {
		w.Header().Add("Content-Type", "application/json")
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(err)
	}
	w.Header().Add("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(res)
}
