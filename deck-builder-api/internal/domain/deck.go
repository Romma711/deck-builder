package domain

import "time"

// Deck es la entidad única de mazo multi-TCG.
//
// GameID/Format referencian el catálogo (games/formats); LeaderCard y
// PartnerCard son referencias a cartas del catálogo (normalizadas en la Fase 4).
//
// Tenancy (Fase 5):
//   - OrgID: "" = mazo personal; con ID = mazo de organización.
//   - IsPrivate: privado = solo visible para su dueño y, si es de org, para
//     los miembros de esa org. Público = visible para cualquiera. El flag se
//     hace cumplir en PostgreSQL vía Row-Level Security.
type Deck struct {
	ID          string
	OwnerID     string
	GameID      string
	GameCode    string // "mtg"... (denormalizado en lecturas via JOIN)
	OrgID       string // "" = personal
	IsPrivate   bool
	Name        string
	Description string
	DeckSize    int
	Format      string // código del formato DENTRO del juego ("commander")
	LeaderCard  *Card
	PartnerCard *Card
	Cards       []CardEntry
	CreatedAt   time.Time
	UpdatedAt   time.Time
}
