package auth

import (
	"github.com/Romma711/deck-builder/internal/ports"
	"golang.org/x/crypto/bcrypt"
)

// PasswordHasher implementa ports.PasswordHasher con bcrypt.
type PasswordHasher struct {
	cost int
}

var _ ports.PasswordHasher = (*PasswordHasher)(nil)

// NewPasswordHasher: cost <= 0 usa el costo por defecto de bcrypt (10).
func NewPasswordHasher(cost int) *PasswordHasher {
	if cost <= 0 {
		cost = bcrypt.DefaultCost
	}
	return &PasswordHasher{cost: cost}
}

func (h *PasswordHasher) Hash(plain string) (string, error) {
	// bcrypt trunca a 72 bytes; validamos antes para no perder silenciosamente.
	if len(plain) > 72 {
		plain = plain[:72]
	}
	hash, err := bcrypt.GenerateFromPassword([]byte(plain), h.cost)
	if err != nil {
		return "", err
	}
	return string(hash), nil
}

func (h *PasswordHasher) Compare(hash, plain string) bool {
	return bcrypt.CompareHashAndPassword([]byte(hash), []byte(plain)) == nil
}
