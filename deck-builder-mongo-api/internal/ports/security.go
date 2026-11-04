package ports

import "github.com/Romma711/deck-builder/internal/domain"

// AccessTokenClaims son los claims verificados de un access token.
type AccessTokenClaims struct {
	UserID   string
	Username string
}

// TokenService emite y verifica access tokens (JWT). La implementación debe
// rechazar tokens con algoritmo distinto al esperado, firmados con otra
// clave, con issuer distinto o vencidos, devolviendo domain.ErrUnauthorized.
type TokenService interface {
	IssueAccessToken(user *domain.User) (string, error)
	ParseAccessToken(token string) (*AccessTokenClaims, error)
}

// PasswordHasher hashea y compara contraseñas (bcrypt, argon2, etc.).
type PasswordHasher interface {
	Hash(plain string) (string, error)
	Compare(hash, plain string) bool
}
