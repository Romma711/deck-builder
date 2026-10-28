package gamespec

import (
	"fmt"

	"github.com/Romma711/deck-builder/internal/domain"
)

// Helpers para leer atributos JSONB de forma segura (JSON numérico llega
// como float64) y acumular errores por campo en *domain.ValidationError.

func attrFloat(attrs map[string]any, key string) (float64, bool) {
	v, exists := attrs[key]
	if !exists {
		return 0, false
	}
	f, ok := v.(float64)
	return f, ok
}

func attrString(attrs map[string]any, key string) (string, bool) {
	v, exists := attrs[key]
	if !exists {
		return "", false
	}
	s, ok := v.(string)
	return s, ok
}

func attrStrings(attrs map[string]any, key string) ([]string, bool) {
	v, exists := attrs[key]
	if !exists {
		return nil, false
	}
	switch list := v.(type) {
	case []string:
		return list, true
	case []any:
		out := make([]string, 0, len(list))
		for _, item := range list {
			s, ok := item.(string)
			if !ok {
				return nil, false
			}
			out = append(out, s)
		}
		return out, true
	default:
		return nil, false
	}
}

// errorCollector acumula errores de varios atributos en un único
// *domain.ValidationError con detalle por campo.
type errorCollector struct {
	details map[string]string
}

func newErrorCollector() *errorCollector {
	return &errorCollector{details: map[string]string{}}
}

func (c *errorCollector) add(field, reason string) {
	c.details[field] = reason
}

// fail devuelve el error acumulado (o nil si todo está bien).
func (c *errorCollector) fail() error {
	if len(c.details) == 0 {
		return nil
	}
	return &domain.ValidationError{Details: c.details}
}

func requireOneof(c *errorCollector, attrs map[string]any, key string, allowed []string) {
	value, ok := attrString(attrs, key)
	if !ok {
		c.add(key, "required")
		return
	}
	for _, a := range allowed {
		if value == a {
			return
		}
	}
	c.add(key, fmt.Sprintf("oneof=%v", allowed))
}

func requireFloatRange(c *errorCollector, attrs map[string]any, key string, min, max float64) {
	value, ok := attrFloat(attrs, key)
	if !ok {
		c.add(key, "required")
		return
	}
	if value < min || value > max {
		c.add(key, fmt.Sprintf("range=[%g,%g]", min, max))
	}
}
