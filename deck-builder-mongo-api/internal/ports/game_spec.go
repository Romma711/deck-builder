package ports

import "github.com/Romma711/deck-builder/internal/domain"

// GameSpec es la estrategia (patrón Strategy) que encapsula las REGLAS de un
// TCG: qué atributos válidos tiene una carta y qué restricciones de mazo
// aplica. Para soportar un juego nuevo basta implementar esta interfaz y
// registrarlo en el Registry (sin tocar los casos de uso).
type GameSpec interface {
	// GameCode identifica al juego que implementa ("mtg", "pokemon"...).
	GameCode() string

	// ValidateCard valida los atributos específicos del juego para una carta
	// (ej.: MTG exige mana_value y type; Pokémon exige hp). Debe devolver
	// *domain.ValidationError con detalles por atributo.
	ValidateCard(card *domain.Card) error

	// ValidateDeck aplica las reglas de mazo del juego más allá de lo que ya
	// cubre el formato (cantidad de copias máximas, límites por rareza, etc.).
	// El servicio YA validó: formato soportado, tamaño contra MinSize/MaxSize
	// y presencia de líder si RequiresLeader.
	ValidateDeck(deck *domain.Deck, format *domain.Format) error
}
