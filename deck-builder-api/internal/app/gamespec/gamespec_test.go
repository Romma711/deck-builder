package gamespec

import (
	"errors"
	"testing"

	"github.com/Romma711/deck-builder/internal/domain"
)

func card(attrs map[string]any) *domain.Card {
	return &domain.Card{
		Name: "Carta", SerialSet: "SET-1", Image: "https://x.dev/c.png",
		Attributes: attrs,
	}
}

func TestRegistry_DuplicadoYVacio(t *testing.T) {
	if _, err := NewRegistry(MTGSpec{}, MTGSpec{}); err == nil {
		t.Error("un spec duplicado debió fallar")
	}
	if _, err := NewRegistry(); err != nil {
		t.Errorf("un registry vacío es válido para arrancar: %v", err)
	}

	r, err := NewRegistry(MTGSpec{}, PokemonSpec{}, YuGiOhSpec{}, OnePieceSpec{})
	if err != nil {
		t.Fatalf("NewRegistry falló: %v", err)
	}
	for _, code := range []string{"mtg", "pokemon", "yuigoh", "onepiece"} {
		if _, ok := r.Spec(code); !ok {
			t.Errorf("falta el spec para %q", code)
		}
	}
	if _, ok := r.Spec("lorcana"); ok {
		t.Error("lorcana no debería tener spec")
	}
}

func TestMTG_ValidateCard(t *testing.T) {
	cases := []struct {
		name  string
		attrs map[string]any
		ok    bool
	}{
		{"válida", map[string]any{"colors": []any{"Blue"}, "mana_value": 3.0, "type": "Creature"}, true},
		{"mana_value fuera de rango", map[string]any{"colors": []any{"Blue"}, "mana_value": 21.0, "type": "Land"}, false},
		{"type inválido", map[string]any{"colors": []any{"Blue"}, "mana_value": 1.0, "type": "Tribal"}, false},
		{"color inválido", map[string]any{"colors": []any{"Rainbow"}, "mana_value": 1.0, "type": "Instant"}, false},
		{"sin colors", map[string]any{"mana_value": 1.0, "type": "Instant"}, false},
		{"faltan atributos", map[string]any{}, false},
		{"mana_value no numérico", map[string]any{"colors": []any{"Red"}, "mana_value": "tres", "type": "Sorcery"}, false},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			err := MTGSpec{}.ValidateCard(card(tc.attrs))
			if tc.ok && err != nil {
				t.Errorf("esperaba OK, obtuve %v", err)
			}
			if !tc.ok {
				var ve *domain.ValidationError
				if !errors.As(err, &ve) {
					t.Errorf("esperaba *domain.ValidationError, obtuve %v", err)
				}
			}
		})
	}
}

func TestMTG_ValidateDeck_Copias(t *testing.T) {
	spec := MTGSpec{}
	format := &domain.Format{Code: "commander"}

	okDeck := &domain.Deck{Cards: []domain.CardEntry{
		{Card: domain.Card{Name: "Forest"}, Quantity: 20}, // básica exenta
		{Card: domain.Card{Name: "Sol Ring"}, Quantity: 4},
	}}
	if err := spec.ValidateDeck(okDeck, format); err != nil {
		t.Errorf("deck válido rechazado: %v", err)
	}

	badDeck := &domain.Deck{Cards: []domain.CardEntry{
		{Card: domain.Card{Name: "Sol Ring"}, Quantity: 5},
	}}
	err := spec.ValidateDeck(badDeck, format)
	var ve *domain.ValidationError
	if !errors.As(err, &ve) {
		t.Fatalf("esperaba *domain.ValidationError, obtuve %v", err)
	}
	if ve.Details["cards.Sol Ring"] == "" {
		t.Errorf("detalles = %v, esperaba cards.Sol Ring", ve.Details)
	}
}

func TestPokemon_ValidateCard(t *testing.T) {
	cases := []struct {
		name  string
		attrs map[string]any
		ok    bool
	}{
		{"pokémon válida", map[string]any{"kind": "pokemon", "hp": 120.0, "energy_type": "Lightning"}, true},
		{"trainer sin hp", map[string]any{"kind": "trainer"}, true},
		{"energy sin hp", map[string]any{"kind": "energy", "energy_type": "Fire"}, true},
		{"hp fuera de rango", map[string]any{"kind": "pokemon", "hp": 400.0, "energy_type": "Water"}, false},
		{"kind inválido", map[string]any{"kind": "amuleto"}, false},
		{"pokemon sin energy_type", map[string]any{"kind": "pokemon", "hp": 60.0}, false},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			err := PokemonSpec{}.ValidateCard(card(tc.attrs))
			if tc.ok && err != nil {
				t.Errorf("esperaba OK, obtuve %v", err)
			}
			if !tc.ok && err == nil {
				t.Error("esperaba error, obtuve OK")
			}
		})
	}
}

func TestPokemon_EnergiasExentasDeLimite(t *testing.T) {
	spec := PokemonSpec{}
	format := &domain.Format{Code: "standard"}

	deck := &domain.Deck{Cards: []domain.CardEntry{
		{Card: domain.Card{Name: "Pikachu", Attributes: map[string]any{"kind": "pokemon"}}, Quantity: 4},
		{Card: domain.Card{Name: "Lightning Energy", Attributes: map[string]any{"kind": "energy"}}, Quantity: 20},
	}}
	if err := spec.ValidateDeck(deck, format); err != nil {
		t.Errorf("deck válido rechazado: %v", err)
	}

	deck.Cards[0].Quantity = 5
	if err := spec.ValidateDeck(deck, format); err == nil {
		t.Error("5 pikachus debieron fallar")
	}
}

func TestYuGiOh_ValidateCard(t *testing.T) {
	cases := []struct {
		name  string
		attrs map[string]any
		ok    bool
	}{
		{"monstruo válido", map[string]any{"card_type": "Monster", "atk": 3000.0, "def": 2500.0}, true},
		{"hechizo válido", map[string]any{"card_type": "Spell"}, true},
		{"monstruo sin atk", map[string]any{"card_type": "Monster", "def": 100.0}, false},
		{"card_type inválido", map[string]any{"card_type": "Trap Card"}, false},
		{"atk negativo", map[string]any{"card_type": "Monster", "atk": -1.0, "def": 0.0}, false},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			err := YuGiOhSpec{}.ValidateCard(card(tc.attrs))
			if tc.ok && err != nil {
				t.Errorf("esperaba OK, obtuve %v", err)
			}
			if !tc.ok && err == nil {
				t.Error("esperaba error, obtuve OK")
			}
		})
	}
}

func TestYuGiOh_TresCopiasMax(t *testing.T) {
	spec := YuGiOhSpec{}
	format := &domain.Format{Code: "master"}

	deck := &domain.Deck{Cards: []domain.CardEntry{
		{Card: domain.Card{Name: "Blue-Eyes White Dragon"}, Quantity: 3},
	}}
	if err := spec.ValidateDeck(deck, format); err != nil {
		t.Errorf("deck válido rechazado: %v", err)
	}
	deck.Cards[0].Quantity = 4
	if err := spec.ValidateDeck(deck, format); err == nil {
		t.Error("4 copias debieron fallar")
	}
}

func TestOnePiece_ValidateCard(t *testing.T) {
	cases := []struct {
		name  string
		attrs map[string]any
		ok    bool
	}{
		{"líder válido", map[string]any{"card_type": "Leader", "life": 5.0}, true},
		{"personaje válido", map[string]any{"card_type": "Character", "power": 5000.0}, true},
		{"evento sin power", map[string]any{"card_type": "Event"}, true},
		{"líder sin life", map[string]any{"card_type": "Leader"}, false},
		{"personaje sin power", map[string]any{"card_type": "Character"}, false},
		{"card_type inválido", map[string]any{"card_type": "Ship"}, false},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			err := OnePieceSpec{}.ValidateCard(card(tc.attrs))
			if tc.ok && err != nil {
				t.Errorf("esperaba OK, obtuve %v", err)
			}
			if !tc.ok && err == nil {
				t.Error("esperaba error, obtuve OK")
			}
		})
	}
}

func TestOnePiece_LiderDebeSerTipoLeader(t *testing.T) {
	spec := OnePieceSpec{}
	format := &domain.Format{Code: "standard", RequiresLeader: true}

	deck := &domain.Deck{LeaderCard: card(map[string]any{"card_type": "Character", "power": 1000.0})}
	err := spec.ValidateDeck(deck, format)
	var ve *domain.ValidationError
	if !errors.As(err, &ve) {
		t.Fatalf("esperaba *domain.ValidationError, obtuve %v", err)
	}
	if ve.Details["leader.card_type"] == "" {
		t.Errorf("detalles = %v, esperaba leader.card_type", ve.Details)
	}

	deck.LeaderCard = card(map[string]any{"card_type": "Leader", "life": 5.0})
	if err := spec.ValidateDeck(deck, format); err != nil {
		t.Errorf("líder válido rechazado: %v", err)
	}
}
