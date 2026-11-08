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

// En One Piece el líder va aparte de las cartas del mazo.
func (OnePieceSpec) LeaderCountsTowardSize() bool { return false }

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

// ValidateDeck: el líder debe ser de tipo "Leader" y hay máximo 4 copias por
// número de carta (distintas cartas pueden compartir nombre, como Luffy).
func (OnePieceSpec) ValidateDeck(deck *domain.Deck, format *domain.Format) error {
	quantities := map[string]int{}
	for _, entry := range deck.Cards {
		quantities[entry.Card.SerialSet] += entry.Quantity
	}
	c := newErrorCollector()
	for serial, qty := range quantities {
		if qty > 4 {
			c.add("cards."+serial, "max=4_copies")
		}
	}
	if err := c.fail(); err != nil {
		return err
	}

	if format.RequiresLeader && deck.LeaderCard != nil {
		if kind, _ := attrString(deck.LeaderCard.Attributes, "card_type"); kind != "Leader" {
			return &domain.ValidationError{Details: map[string]string{
				"leader.card_type": fmt.Sprintf("oneof=[Leader] (obligatorio para %s)", format.Code),
			}}
		}
	}
	return nil
}
