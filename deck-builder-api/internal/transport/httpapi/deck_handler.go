package httpapi

import (
	"net/http"
	"time"

	deckapp "github.com/Romma711/deck-builder/internal/app/deck"
	"github.com/Romma711/deck-builder/internal/domain"
	"github.com/Romma711/deck-builder/internal/pkg/validation"
	"github.com/Romma711/deck-builder/internal/ports"
	"github.com/gin-gonic/gin"
)

type deckHandler struct {
	svc *deckapp.Service
}

// RegisterDeckRoutes monta /decks de la API v3:
//
//	GET    /decks            (público, filtros ?game=&owner=&org=&format=&leader=)
//	GET    /decks/:id        (público; RLS decide: privado ajeno → 404)
//	POST   /decks            (autenticado)
//	PUT    /decks/:id        (autenticado, solo el dueño)
//	DELETE /decks/:id        (autenticado, solo el dueño)
//
// Las lecturas usan AuthOptional: sin token siguen siendo públicas, pero CON
// token el contexto lleva el user ID y RLS permite ver los mazos privados
// propios (y los de mis orgs) en los mismos endpoints.
func RegisterDeckRoutes(rg *gin.RouterGroup, svc *deckapp.Service, tokens ports.TokenService) {
	h := &deckHandler{svc: svc}

	rg.GET("/decks", AuthOptional(tokens), h.list)
	rg.GET("/decks/:id", AuthOptional(tokens), h.get)

	authed := rg.Group("/decks", Auth(tokens))
	authed.POST("", h.create)
	authed.PUT("/:id", h.update)
	authed.DELETE("/:id", h.delete)
}

// --- request/response ---
// DTOs genéricos multi-TCG: los atributos específicos de cada juego
// (mana_value de MTG, hp de Pokémon...) viajan en "attributes" y su esquema
// lo valida el GameSpec correspondiente en la capa de aplicación.

type cardRequest struct {
	Name       string         `json:"name" validate:"required,min=1,max=100"`
	SerialSet  string         `json:"serial_set" validate:"required,min=1,max=50"`
	Image      string         `json:"image" validate:"required,url"`
	Attributes map[string]any `json:"attributes" validate:"required"`
}

type cardEntryRequest struct {
	Card     cardRequest `json:"card"`
	Quantity int         `json:"quantity" validate:"required,min=1,max=99"`
}

type deckRequest struct {
	Name        string             `json:"name" validate:"required,min=3,max=50"`
	Description string             `json:"description" validate:"omitempty,max=300"`
	DeckSize    int                `json:"deck_size" validate:"required,min=1,max=400"`
	Game        string             `json:"game" validate:"required"`
	Format      string             `json:"format" validate:"required"`
	OrgID       string             `json:"org_id" validate:"omitempty,uuid"` // "" = personal
	Private     bool               `json:"private"`                          // solo dueño/miembros de su org
	Leader      *cardRequest       `json:"leader,omitempty"`
	Partner     *cardRequest       `json:"partner,omitempty"`
	Cards       []cardEntryRequest `json:"cards" validate:"required,min=1,dive"`
}

type cardResponse struct {
	ID         string         `json:"id,omitempty"`
	Name       string         `json:"name"`
	SerialSet  string         `json:"serial_set"`
	Image      string         `json:"image"`
	Attributes map[string]any `json:"attributes"`
}

type cardEntryResponse struct {
	Card     cardResponse `json:"card"`
	Quantity int          `json:"quantity"`
}

type deckResponse struct {
	ID          string              `json:"id"`
	OwnerID     string              `json:"owner_id"`
	OrgID       string              `json:"org_id,omitempty"` // vacío = mazo personal
	Private     bool                `json:"private"`
	Name        string              `json:"name"`
	Description string              `json:"description"`
	DeckSize    int                 `json:"deck_size"`
	Game        string              `json:"game"`
	Format      string              `json:"format"`
	Leader      *cardResponse       `json:"leader,omitempty"`
	Partner     *cardResponse       `json:"partner,omitempty"`
	Cards       []cardEntryResponse `json:"cards,omitempty"`
	CreatedAt   time.Time           `json:"created_at"`
	UpdatedAt   time.Time           `json:"updated_at"`
}

func newCardResponse(card *domain.Card) *cardResponse {
	if card == nil {
		return nil
	}
	return &cardResponse{
		ID:         card.ID,
		Name:       card.Name,
		SerialSet:  card.SerialSet,
		Image:      card.Image,
		Attributes: card.Attributes,
	}
}

func newDeckResponse(deck *domain.Deck) deckResponse {
	resp := deckResponse{
		ID:          deck.ID,
		OwnerID:     deck.OwnerID,
		OrgID:       deck.OrgID,
		Private:     deck.IsPrivate,
		Name:        deck.Name,
		Description: deck.Description,
		DeckSize:    deck.DeckSize,
		Game:        deck.GameCode,
		Format:      deck.Format,
		Leader:      newCardResponse(deck.LeaderCard),
		Partner:     newCardResponse(deck.PartnerCard),
		CreatedAt:   deck.CreatedAt,
		UpdatedAt:   deck.UpdatedAt,
	}
	if len(deck.Cards) > 0 {
		resp.Cards = make([]cardEntryResponse, len(deck.Cards))
		for i, entry := range deck.Cards {
			resp.Cards[i] = cardEntryResponse{
				Card:     *newCardResponse(&entry.Card),
				Quantity: entry.Quantity,
			}
		}
	}
	return resp
}

func (r deckRequest) toDomainInput() deckapp.DeckInput {
	input := deckapp.DeckInput{
		Name:        r.Name,
		Description: r.Description,
		DeckSize:    r.DeckSize,
		GameCode:    r.Game,
		Format:      r.Format,
		OrgID:       r.OrgID,
		IsPrivate:   r.Private,
	}
	if r.Leader != nil {
		input.Leader = r.Leader.toDomainCard()
	}
	if r.Partner != nil {
		input.Partner = r.Partner.toDomainCard()
	}
	for _, entry := range r.Cards {
		input.Cards = append(input.Cards, domain.CardEntry{
			Card:     *entry.Card.toDomainCard(),
			Quantity: entry.Quantity,
		})
	}
	return input
}

func (r cardRequest) toDomainCard() *domain.Card {
	return &domain.Card{
		Name:       r.Name,
		SerialSet:  r.SerialSet,
		Image:      r.Image,
		Attributes: r.Attributes,
	}
}

// --- handlers ---

func (h *deckHandler) create(c *gin.Context) {
	var req deckRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		RespondError(c, &domain.ValidationError{Message: "JSON inválido"})
		return
	}
	if err := validation.Struct(req); err != nil {
		RespondError(c, err)
		return
	}

	user, ok := CurrentUser(c)
	if !ok {
		RespondError(c, domain.ErrUnauthorized)
		return
	}

	deck, err := h.svc.Create(c.Request.Context(), user.ID, req.toDomainInput())
	if err != nil {
		RespondError(c, err)
		return
	}

	c.JSON(http.StatusCreated, newDeckResponse(deck))
}

func (h *deckHandler) get(c *gin.Context) {
	deck, err := h.svc.GetByID(c.Request.Context(), c.Param("id"))
	if err != nil {
		RespondError(c, err)
		return
	}
	c.JSON(http.StatusOK, newDeckResponse(deck))
}

func (h *deckHandler) list(c *gin.Context) {
	decks, err := h.svc.List(c.Request.Context(), ports.DeckFilter{
		GameCode:   c.Query("game"),
		OwnerID:    c.Query("owner"),
		OrgID:      c.Query("org"),
		Format:     c.Query("format"),
		LeaderName: c.Query("leader"),
	})
	if err != nil {
		RespondError(c, err)
		return
	}

	items := make([]deckResponse, 0, len(decks))
	for i := range decks {
		items = append(items, newDeckResponse(&decks[i]))
	}
	c.JSON(http.StatusOK, gin.H{"data": items, "meta": gin.H{"total": len(items)}})
}

func (h *deckHandler) update(c *gin.Context) {
	var req deckRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		RespondError(c, &domain.ValidationError{Message: "JSON inválido"})
		return
	}
	if err := validation.Struct(req); err != nil {
		RespondError(c, err)
		return
	}

	user, ok := CurrentUser(c)
	if !ok {
		RespondError(c, domain.ErrUnauthorized)
		return
	}

	deck, err := h.svc.Update(c.Request.Context(), c.Param("id"), user.ID, req.toDomainInput())
	if err != nil {
		RespondError(c, err)
		return
	}
	c.JSON(http.StatusOK, newDeckResponse(deck))
}

func (h *deckHandler) delete(c *gin.Context) {
	user, ok := CurrentUser(c)
	if !ok {
		RespondError(c, domain.ErrUnauthorized)
		return
	}

	if err := h.svc.Delete(c.Request.Context(), c.Param("id"), user.ID); err != nil {
		RespondError(c, err)
		return
	}
	c.Status(http.StatusNoContent)
}
