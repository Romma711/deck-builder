package httpapi

import (
	"net/http"
	"strconv"

	"github.com/Romma711/deck-builder/internal/domain"
	"github.com/Romma711/deck-builder/internal/ports"
	"github.com/gin-gonic/gin"
)

type catalogHandler struct {
	games ports.GameRepository
	cards ports.CardRepository
}

// RegisterCatalogRoutes monta el catálogo público de la API v3:
//
//	GET /games                 (juegos activos con sus formatos)
//	GET /games/:code           (un juego con sus formatos)
//	GET /cards                 (catálogo, filtros ?game=&q=&limit=&offset=)
//	GET /cards/:id             (una carta del catálogo)
func RegisterCatalogRoutes(rg *gin.RouterGroup, games ports.GameRepository, cards ports.CardRepository) {
	h := &catalogHandler{games: games, cards: cards}

	rg.GET("/games", h.listGames)
	rg.GET("/games/:code", h.getGame)
	rg.GET("/cards", h.listCards)
	rg.GET("/cards/:id", h.getCard)
}

// --- games ---

type formatResponse struct {
	Code           string `json:"code"`
	Name           string `json:"name"`
	DeckSize       int    `json:"deck_size"`
	MinSize        int    `json:"min_size"`
	MaxSize        int    `json:"max_size"`
	RequiresLeader bool   `json:"requires_leader"`
}

type gameResponse struct {
	Code    string           `json:"code"`
	Name    string           `json:"name"`
	Formats []formatResponse `json:"formats"`
}

func newGameResponse(game *domain.Game) gameResponse {
	formats := make([]formatResponse, 0, len(game.Formats))
	for _, f := range game.Formats {
		formats = append(formats, formatResponse{
			Code:           f.Code,
			Name:           f.Name,
			DeckSize:       f.DeckSize,
			MinSize:        f.MinSize,
			MaxSize:        f.MaxSize,
			RequiresLeader: f.RequiresLeader,
		})
	}
	return gameResponse{Code: game.Code, Name: game.Name, Formats: formats}
}

func (h *catalogHandler) listGames(c *gin.Context) {
	games, err := h.games.FindActive(c.Request.Context())
	if err != nil {
		RespondError(c, err)
		return
	}

	items := make([]gameResponse, 0, len(games))
	for i := range games {
		items = append(items, newGameResponse(&games[i]))
	}
	c.JSON(http.StatusOK, gin.H{"data": items, "meta": gin.H{"total": len(items)}})
}

func (h *catalogHandler) getGame(c *gin.Context) {
	game, err := h.games.FindByCode(c.Request.Context(), c.Param("code"))
	if err != nil {
		RespondError(c, err)
		return
	}
	c.JSON(http.StatusOK, newGameResponse(game))
}

// --- cards ---

type cardCatalogResponse struct {
	ID         string         `json:"id"`
	Game       string         `json:"game"`
	Name       string         `json:"name"`
	SerialSet  string         `json:"serial_set"`
	Image      string         `json:"image"`
	Attributes map[string]any `json:"attributes"`
}

func (h *catalogHandler) listCards(c *gin.Context) {
	limit, _ := strconv.Atoi(c.Query("limit"))
	offset, _ := strconv.Atoi(c.Query("offset"))
	if offset < 0 {
		offset = 0
	}

	cards, total, err := h.cards.Find(c.Request.Context(), ports.CardFilter{
		GameCode: c.Query("game"),
		Query:    c.Query("q"),
		Limit:    limit,
		Offset:   offset,
	})
	if err != nil {
		RespondError(c, err)
		return
	}

	// Mapa game_id → code para responder con el código, no el UUID.
	gameCodes := map[string]string{}
	if len(cards) > 0 {
		games, err := h.games.FindActive(c.Request.Context())
		if err != nil {
			RespondError(c, err)
			return
		}
		for _, g := range games {
			gameCodes[g.ID] = g.Code
		}
	}

	items := make([]cardCatalogResponse, 0, len(cards))
	for i := range cards {
		items = append(items, cardCatalogResponse{
			ID:         cards[i].ID,
			Game:       gameCodes[cards[i].GameID],
			Name:       cards[i].Name,
			SerialSet:  cards[i].SerialSet,
			Image:      cards[i].Image,
			Attributes: cards[i].Attributes,
		})
	}

	c.JSON(http.StatusOK, gin.H{
		"data": items,
		"meta": gin.H{"total": total, "limit": limit, "offset": offset},
	})
}

func (h *catalogHandler) getCard(c *gin.Context) {
	card, err := h.cards.FindByID(c.Request.Context(), c.Param("id"))
	if err != nil {
		RespondError(c, err)
		return
	}

	gameCode := ""
	if games, err := h.games.FindActive(c.Request.Context()); err == nil {
		for _, g := range games {
			if g.ID == card.GameID {
				gameCode = g.Code
				break
			}
		}
	}

	c.JSON(http.StatusOK, cardCatalogResponse{
		ID:         card.ID,
		Game:       gameCode,
		Name:       card.Name,
		SerialSet:  card.SerialSet,
		Image:      card.Image,
		Attributes: card.Attributes,
	})
}
