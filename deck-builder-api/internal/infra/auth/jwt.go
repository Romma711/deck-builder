// Package auth implementa los puertos de seguridad concretos:
// JWT (golang-jwt v5) y bcrypt.
package auth

import (
	"fmt"
	"time"

	"github.com/Romma711/deck-builder/internal/domain"
	"github.com/Romma711/deck-builder/internal/ports"
	"github.com/golang-jwt/jwt/v5"
)

const issuer = "deck-builder"

// TokenService emite y verifica access tokens JWT firmados con HS256.
type TokenService struct {
	secret    []byte
	accessTTL time.Duration
}

var _ ports.TokenService = (*TokenService)(nil)

func NewTokenService(secret string, accessTTL time.Duration) *TokenService {
	return &TokenService{secret: []byte(secret), accessTTL: accessTTL}
}

func (s *TokenService) IssueAccessToken(user *domain.User) (string, error) {
	now := time.Now()
	claims := jwt.MapClaims{
		"sub":      user.ID,
		"username": user.Username,
		"iss":      issuer,
		"iat":      now.Unix(),
		"exp":      now.Add(s.accessTTL).Unix(),
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	return token.SignedString(s.secret)
}

// ParseAccessToken valida el token por COMPLETO:
//   - algoritmo debe ser HS256 (WithValidMethods → previene algorithm confusion
//     y el clásico ataque "none")
//   - firma correcta para nuestra clave
//   - issuer correcto
//   - exp presente y no vencido
func (s *TokenService) ParseAccessToken(raw string) (*ports.AccessTokenClaims, error) {
	token, err := jwt.Parse(raw, func(t *jwt.Token) (any, error) {
		return s.secret, nil
	},
		jwt.WithValidMethods([]string{jwt.SigningMethodHS256.Alg()}),
		jwt.WithIssuer(issuer),
		jwt.WithExpirationRequired(),
	)
	if err != nil || !token.Valid {
		return nil, fmt.Errorf("%w: token inválido", domain.ErrUnauthorized)
	}

	mapClaims, ok := token.Claims.(jwt.MapClaims)
	if !ok {
		return nil, fmt.Errorf("%w: claims inválidos", domain.ErrUnauthorized)
	}

	// Type-assertions SEGUROS (sin panic ante claims faltantes).
	userID, ok := mapClaims["sub"].(string)
	if !ok || userID == "" {
		return nil, fmt.Errorf("%w: sub ausente", domain.ErrUnauthorized)
	}
	username, _ := mapClaims["username"].(string)

	return &ports.AccessTokenClaims{UserID: userID, Username: username}, nil
}
