package gamespec

import (
	"fmt"

	"github.com/Romma711/deck-builder/internal/domain"
	"github.com/Romma711/deck-builder/internal/ports"
)

// Spec de One Piece Card Game.
var _ ports.GameSpec = (*OnePieceSpec)(nil)

type OnePieceSpec struct{}

func (OnePieceSpec) GameCode() string { return "onepiece" }

var onepieceCardTypes = []string{"Leader", "Character", "Event", "Stage"}

// ValidateCard: card_type obligatorio; líderes exigen life ≥ 1; personajes
// exigen power [0,12000].
func (OnePieceSpec) ValidateCard(card *domain.Card) error {
	c := newErrorCollector()
	requireOneof(c, card.Attributes, "card_type", onepieceCardTypes)

	cardType, _ := attrString(card.Attributes, "card_type")
	switch cardType {
	case "Leader":
		requireFloatRange(c, card.Attributes, "life", 1, 10)
	case "Character":
		requireFloatRange(c, card.Attributes, "power", 0, 12000)
	}
	return c.fail()
}

// ValidateDeck: el líder del mazo debe ser de tipo "Leader".
// One Piece no impone límite de copias por carta en su reglamento actual.
func (OnePieceSpec) ValidateDeck(deck *domain.Deck, format *domain.Format) error {
	if format.RequiresLeader && deck.LeaderCard != nil {
		if kind, _ := attrString(deck.LeaderCard.Attributes, "card_type"); kind != "Leader" {
			return &domain.ValidationError{Details: map[string]string{
				"leader.card_type": fmt.Sprintf("oneof=[Leader] (obligatorio para %s)", format.Code),
			}}
		}
	}
	return nil
}
