package domain

// Game es un TCG soportado (MTG, Pokémon, Yu-Gi-Oh!, One Piece...).
// Los juegos y formatos se persisten en PostgreSQL (sembrados por migración).
type Game struct {
	ID      string
	Code    string // "mtg", "pokemon", "yuigoh", "onepiece"
	Name    string
	Active  bool
	Formats []Format
}

// FormatByCode devuelve el formato del juego (case-insensitive) o nil.
func (g *Game) FormatByCode(code string) *Format {
	for i := range g.Formats {
		if g.Formats[i].Code == code {
			return &g.Formats[i]
		}
	}
	return nil
}

// Format son las reglas de tamaño/líder de un formato dentro de un juego.
type Format struct {
	ID             string
	GameID         string
	Code           string
	Name           string
	DeckSize       int  // tamaño típico (informativo)
	MinSize        int  // 0 = sin mínimo
	MaxSize        int  // 0 = sin máximo
	RequiresLeader bool // ej.: commander (MTG), STANDARD (One Piece)
}

// AllowsSize valida un tamaño de mazo contra los límites del formato.
func (f Format) AllowsSize(size int) bool {
	if f.MinSize > 0 && size < f.MinSize {
		return false
	}
	if f.MaxSize > 0 && size > f.MaxSize {
		return false
	}
	return true
}
