package postgres

import (
	"encoding/json"
	"fmt"

	"github.com/Romma711/deck-builder/internal/domain"
	"gorm.io/datatypes"
)

// Helpers de la columna attributes (JSONB): serialización del modelo híbrido
// columnas comunes + atributos específicos de cada juego TCG.

func marshalAttributes(attrs map[string]any) (datatypes.JSON, error) {
	if attrs == nil {
		return datatypes.JSON("{}"), nil
	}
	data, err := json.Marshal(attrs)
	if err != nil {
		return nil, fmt.Errorf("%w: serializar atributos: %v", domain.ErrInternal, err)
	}
	return datatypes.JSON(data), nil
}

func unmarshalAttributes(data datatypes.JSON) (map[string]any, error) {
	if len(data) == 0 || string(data) == "null" {
		return map[string]any{}, nil
	}
	var attrs map[string]any
	if err := json.Unmarshal(data, &attrs); err != nil {
		return nil, fmt.Errorf("%w: deserializar atributos: %v", domain.ErrInternal, err)
	}
	return attrs, nil
}
