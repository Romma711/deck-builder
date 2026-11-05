package domain

import "time"

// RefreshToken representa un refresh token GUARDADO (el valor en claro solo
// existe en el cliente; aquí vive únicamente su hash SHA-256).
type RefreshToken struct {
	ID        string
	UserID    string
	TokenHash string
	ExpiresAt time.Time
	RevokedAt *time.Time
	CreatedAt time.Time
}

// IsExpired informa si el token venció respecto al instante dado.
func (t *RefreshToken) IsExpired(now time.Time) bool {
	return now.After(t.ExpiresAt)
}

// IsValid: no vencido y no revocado.
func (t *RefreshToken) IsValid(now time.Time) bool {
	return t.RevokedAt == nil && !t.IsExpired(now)
}
