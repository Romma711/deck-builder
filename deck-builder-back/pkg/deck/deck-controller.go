package deck

import(
	"deck-builder-back/pkg/controller/middlewere"
	"deck-builder-back/pkg/types"
	"encoding/json"
	"net/http"
	"strconv"

	"github.com/gorilla/mux"
)

type Handler struct{
	store types.DeckStore
}

func NewHandler(store types.DeckStore)