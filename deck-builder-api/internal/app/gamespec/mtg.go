package gamespec

import (
	"fmt"

	"github.com/Romma711/deck-builder/internal/domain"
	"github.com/Romma711/deck-builder/internal/ports"
)

// Spec de Magic: The Gathering.
var _ ports.GameSpec = (*MTGSpec)(nil)

type MTGSpec struct{}

func (MTGSpec) GameCode() string { return "mtg" }

// En Commander el comandante es parte de las 100 cartas.
func (MTGSpec) LeaderCountsTowardSize() bool { return true }

var (
	mtgColors = []string{"White", "Blue", "Black", "Red", "Green", "Colorless"}
	mtgTypes  = []string{"Creature", "Instant", "Sorcery", "Artifact", "Enchantment", "Land", "Planeswalker"}
	// Básicas exentas del límite de 4 copias.
	mtgBasicLands = map[string]bool{
		"Plains": true, "Island": true, "Swamp": true,
		"Mountain": true, "Forest": true, "Wastes": true,
	}
	// Formatos singleton: una sola copia por carta (básicas exentas).
	mtgSingletonFormats = map[string]bool{"commander": true, "brawl": true}
)

// ValidateCard: mana_value [0,20], type en la lista MTG, colors subconjunto.
func (MTGSpec) ValidateCard(card *domain.Card) error {
	c := newErrorCollector()
	requireFloatRange(c, card.Attributes, "mana_value", 0, 20)
	requireOneof(c, card.Attributes, "type", mtgTypes)

	colors, ok := attrStrings(card.Attributes, "colors")
	if !ok || len(colors) == 0 {
		c.add("colors", "required")
	} else {
		for _, color := range colors {
			valid := false
			for _, allowed := range mtgColors {
				if color == allowed {
					valid = true
					break
				}
			}
			if !valid {
				c.add("colors", fmt.Sprintf("oneof=%v", mtgColors))
				break
			}
		}
	}
	return c.fail()
}

// ValidateDeck: máximo 4 copias por nombre, 1 en formatos singleton (básicas exentas).
func (MTGSpec) ValidateDeck(deck *domain.Deck, format *domain.Format) error {
	limit := 4
	if format != nil && mtgSingletonFormats[format.Code] {
		limit = 1
	}

	quantities := map[string]int{}
	for _, entry := range deck.Cards {
		quantities[entry.Card.Name] += entry.Quantity
	}

	c := newErrorCollector()
	for name, qty := range quantities {
		if mtgBasicLands[name] {
			continue
		}
		if qty > limit {
			c.add("cards."+name, fmt.Sprintf("max=%d_copies", limit))
		}
	}
	return c.fail()
}
