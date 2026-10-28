package gamespec

import (
	"github.com/Romma711/deck-builder/internal/domain"
	"github.com/Romma711/deck-builder/internal/ports"
)

// Spec de Yu-Gi-Oh!.
var _ ports.GameSpec = (*YuGiOhSpec)(nil)

type YuGiOhSpec struct{}

func (YuGiOhSpec) GameCode() string { return "yuigoh" }

var yuigohCardTypes = []string{"Monster", "Spell", "Trap"}

// ValidateCard: card_type obligatorio; los monstruos exigen atk/def ≥ 0.
func (YuGiOhSpec) ValidateCard(card *domain.Card) error {
	c := newErrorCollector()
	requireOneof(c, card.Attributes, "card_type", yuigohCardTypes)

	cardType, _ := attrString(card.Attributes, "card_type")
	if cardType == "Monster" {
		requireFloatRange(c, card.Attributes, "atk", 0, 9999)
		requireFloatRange(c, card.Attributes, "def", 0, 9999)
	}
	return c.fail()
}

// ValidateDeck: máximo 3 copias por nombre (límite estándar del juego).
func (YuGiOhSpec) ValidateDeck(deck *domain.Deck, _ *domain.Format) error {
	quantities := map[string]int{}
	for _, entry := range deck.Cards {
		quantities[entry.Card.Name] += entry.Quantity
	}

	c := newErrorCollector()
	for name, qty := range quantities {
		if qty > 3 {
			c.add("cards."+name, "max=3_copies")
		}
	}
	return c.fail()
}
