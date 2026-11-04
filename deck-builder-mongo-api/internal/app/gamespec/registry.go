// Package gamespec implementa las estrategias de validación por juego TCG
// (patrón Strategy + Registry). Agregar un juego nuevo = implementar
// ports.GameSpec y registrarlo en cmd/api/main.go.
package gamespec

import (
	"fmt"

	"github.com/Romma711/deck-builder/internal/ports"
)

// Registry indexa los GameSpec por código de juego.
type Registry struct {
	specs map[string]ports.GameSpec
}

// NewRegistry construye el registro; falla si hay specs sin código o duplicados
// (falla al arrancar, no en runtime).
func NewRegistry(specs ...ports.GameSpec) (*Registry, error) {
	r := &Registry{specs: make(map[string]ports.GameSpec, len(specs))}
	for _, spec := range specs {
		code := spec.GameCode()
		if code == "" {
			return nil, fmt.Errorf("GameSpec sin código")
		}
		if _, exists := r.specs[code]; exists {
			return nil, fmt.Errorf("GameSpec duplicado para el juego %q", code)
		}
		r.specs[code] = spec
	}
	return r, nil
}

// Spec devuelve la estrategia del juego o false si no está soportado.
func (r *Registry) Spec(gameCode string) (ports.GameSpec, bool) {
	spec, ok := r.specs[gameCode]
	return spec, ok
}
