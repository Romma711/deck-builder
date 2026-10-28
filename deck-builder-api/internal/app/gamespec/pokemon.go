package gamespec

import (
	"github.com/Romma711/deck-builder/internal/domain"
	"github.com/Romma711/deck-builder/internal/ports"
)

// Spec de Pokémon TCG.
var _ ports.GameSpec = (*PokemonSpec)(nil)

type PokemonSpec struct{}

func (PokemonSpec) GameCode() string { return "pokemon" }

var (
	pokemonKinds      = []string{"pokemon", "trainer", "energy"}
	pokemonEnergyType = []string{
		"Lightning", "Water", "Grass", "Fire", "Psychic",
		"Fighting", "Darkness", "Metal", "Dragon", "Colorless",
	}
)

// ValidateCard: kind obligatorio; si es pokemon → hp [1,350] y energy_type.
func (PokemonSpec) ValidateCard(card *domain.Card) error {
	c := newErrorCollector()
	requireOneof(c, card.Attributes, "kind", pokemonKinds)

	kind, _ := attrString(card.Attributes, "kind")
	if kind == "pokemon" {
		requireFloatRange(c, card.Attributes, "hp", 1, 350)
		requireOneof(c, card.Attributes, "energy_type", pokemonEnergyType)
	}
	return c.fail()
}

// ValidateDeck: máximo 4 copias por nombre; las energías quedan exentas
// (las básicas son ilimitadas en el reglamento).
func (PokemonSpec) ValidateDeck(deck *domain.Deck, _ *domain.Format) error {
	quantities := map[string]int{}
	energyNames := map[string]bool{}
	for _, entry := range deck.Cards {
		quantities[entry.Card.Name] += entry.Quantity
		if kind, _ := attrString(entry.Card.Attributes, "kind"); kind == "energy" {
			energyNames[entry.Card.Name] = true
		}
	}

	c := newErrorCollector()
	for name, qty := range quantities {
		if energyNames[name] {
			continue
		}
		if qty > 4 {
			c.add("cards."+name, "max=4_copies")
		}
	}
	return c.fail()
}
