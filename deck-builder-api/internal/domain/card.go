package domain

// Card es una carta del catálogo multi-TCG.
//
// Modelo HÍBRIDO (decisión de la Fase 4):
//   - Columnas comunes consultables: Name, SerialSet, Image.
//   - Attributes (JSONB): lo específico de cada juego — mana_value/colors de
//     MTG, hp/energy_type de Pokémon, atk/def de Yu-Gi-Oh, power/type de
//     One Piece. Su esquema lo define el GameSpec de cada juego.
//
// Los tags json definen el esquema persistido en la columna attributes.
type Card struct {
	ID         string
	GameID     string
	Name       string
	SerialSet  string
	Image      string
	Attributes map[string]any
}

// CardEntry es una carta del mazo con su cantidad.
type CardEntry struct {
	Card     Card `json:"card"`
	Quantity int  `json:"quantity"`
}
