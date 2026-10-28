// Package auth implementa el caso de uso de autenticación: registro, login,
// refresh con rotación y logout. Depende SOLO de los puertos (interfaces),
// nunca de la infraestructura concreta: es 100% testeable con fakes.
package auth

import (
	"context"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/Romma711/deck-builder/internal/domain"
	"github.com/Romma711/deck-builder/internal/ports"
)

// MinPasswordLength define el mínimo de la contraseña.
const MinPasswordLength = 8

type Service struct {
	users       ports.UserRepository
	refreshRepo ports.RefreshTokenRepository
	tokens      ports.TokenService
	hasher      ports.PasswordHasher
	accessTTL   time.Duration
	refreshTTL  time.Duration
	now         func() time.Time // inyectable para tests
}

func NewService(
	users ports.UserRepository,
	refreshRepo ports.RefreshTokenRepository,
	tokens ports.TokenService,
	hasher ports.PasswordHasher,
	accessTTL time.Duration,
	refreshTTL time.Duration,
) *Service {
	return &Service{
		users:       users,
		refreshRepo: refreshRepo,
		tokens:      tokens,
		hasher:      hasher,
		accessTTL:   accessTTL,
		refreshTTL:  refreshTTL,
		now:         func() time.Time { return time.Now().UTC() },
	}
}

// --- DTOs de entrada/salida del caso de uso ---

type RegisterInput struct {
	Username string
	Email    string
	Password string
}

type UpdateProfileInput struct {
	Username string
	Email    string
}

type TokenPair struct {
	AccessToken  string
	RefreshToken string
	ExpiresIn    int64 // segundos del access token
}

// --- Registro ---

func (s *Service) Register(ctx context.Context, in RegisterInput) (*domain.User, error) {
	if err := validateRegistration(in); err != nil {
		return nil, err
	}

	hash, err := s.hasher.Hash(in.Password)
	if err != nil {
		return nil, fmt.Errorf("%w: hashear contraseña: %v", domain.ErrInternal, err)
	}

	now := s.now()
	user := &domain.User{
		Username:     strings.TrimSpace(in.Username),
		Email:        normalizeEmail(in.Email),
		PasswordHash: hash,
		CreatedAt:    now,
		UpdatedAt:    now,
	}

	if err := s.users.Create(ctx, user); err != nil {
		return nil, err // ErrConflict si el username/email ya existe
	}
	return user, nil
}

// --- Login ---

func (s *Service) Login(ctx context.Context, username, password string) (*TokenPair, error) {
	user, err := s.users.FindByUsername(ctx, strings.TrimSpace(username))
	if err != nil {
		// Mismo error genérico para "no existe" y "contraseña incorrecta":
		// evita enumeración de usuarios.
		if errors.Is(err, domain.ErrNotFound) {
			return nil, domain.ErrUnauthorized
		}
		return nil, err
	}

	if !s.hasher.Compare(user.PasswordHash, password) {
		return nil, domain.ErrUnauthorized
	}

	return s.issuePair(ctx, user)
}

// --- Refresh con rotación ---

func (s *Service) Refresh(ctx context.Context, rawToken string) (*TokenPair, error) {
	if rawToken == "" {
		return nil, domain.ErrUnauthorized
	}

	stored, err := s.refreshRepo.FindByHash(ctx, hashToken(rawToken))
	if err != nil {
		if errors.Is(err, domain.ErrNotFound) {
			return nil, domain.ErrUnauthorized
		}
		return nil, err
	}

	// Reuso de un token revocado = señal de robo → revocamos toda la familia.
	if stored.RevokedAt != nil {
		_ = s.refreshRepo.RevokeAllForUser(ctx, stored.UserID)
		return nil, domain.ErrUnauthorized
	}

	if stored.IsExpired(s.now()) {
		return nil, domain.ErrUnauthorized
	}

	if err := s.refreshRepo.Revoke(ctx, stored.ID); err != nil {
		return nil, err
	}

	user, err := s.users.FindByID(ctx, stored.UserID)
	if err != nil {
		return nil, err
	}

	return s.issuePair(ctx, user)
}

// --- Logout (idempotente) ---

func (s *Service) Logout(ctx context.Context, rawToken string) error {
	if rawToken == "" {
		return nil
	}
	stored, err := s.refreshRepo.FindByHash(ctx, hashToken(rawToken))
	if err != nil {
		if errors.Is(err, domain.ErrNotFound) {
			return nil
		}
		return err
	}
	return s.refreshRepo.Revoke(ctx, stored.ID)
}

// --- Perfil ---

func (s *Service) Me(ctx context.Context, userID string) (*domain.User, error) {
	return s.users.FindByID(ctx, userID)
}

func (s *Service) UpdateProfile(ctx context.Context, userID string, in UpdateProfileInput) (*domain.User, error) {
	user, err := s.users.FindByID(ctx, userID)
	if err != nil {
		return nil, err
	}
	user.Username = strings.TrimSpace(in.Username)
	user.Email = normalizeEmail(in.Email)
	user.UpdatedAt = s.now()

	if err := s.users.Update(ctx, user); err != nil {
		return nil, err
	}
	return user, nil
}

func (s *Service) DeleteAccount(ctx context.Context, userID string) error {
	if err := s.users.Delete(ctx, userID); err != nil {
		return err
	}
	// Invalidamos cualquier refresh token vigente del usuario.
	return s.refreshRepo.RevokeAllForUser(ctx, userID)
}

// --- internos ---

func (s *Service) issuePair(ctx context.Context, user *domain.User) (*TokenPair, error) {
	access, err := s.tokens.IssueAccessToken(user)
	if err != nil {
		return nil, fmt.Errorf("%w: emitir access token: %v", domain.ErrInternal, err)
	}

	rawRefresh, refresh, err := s.newRefreshToken(user.ID)
	if err != nil {
		return nil, err
	}

	if err := s.refreshRepo.Save(ctx, refresh); err != nil {
		return nil, fmt.Errorf("%w: guardar refresh token: %v", domain.ErrInternal, err)
	}

	return &TokenPair{
		AccessToken:  access,
		RefreshToken: rawRefresh,
		ExpiresIn:    int64(s.accessTTL.Seconds()),
	}, nil
}

func (s *Service) newRefreshToken(userID string) (string, *domain.RefreshToken, error) {
	buf := make([]byte, 32)
	if _, err := rand.Read(buf); err != nil {
		return "", nil, fmt.Errorf("%w: generar refresh token: %v", domain.ErrInternal, err)
	}
	raw := base64.RawURLEncoding.EncodeToString(buf)

	now := s.now()
	return raw, &domain.RefreshToken{
		UserID:    userID,
		TokenHash: hashToken(raw),
		ExpiresAt: now.Add(s.refreshTTL),
		CreatedAt: now,
	}, nil
}

func validateRegistration(in RegisterInput) error {
	details := map[string]string{}
	if strings.TrimSpace(in.Username) == "" {
		details["Username"] = "required"
	}
	if !strings.Contains(in.Email, "@") {
		details["Email"] = "email"
	}
	if len(in.Password) < MinPasswordLength {
		details["Password"] = fmt.Sprintf("min=%d", MinPasswordLength)
	}
	if len(details) > 0 {
		return &domain.ValidationError{Details: details}
	}
	return nil
}

func hashToken(raw string) string {
	sum := sha256.Sum256([]byte(raw))
	return hex.EncodeToString(sum[:])
}

func normalizeEmail(email string) string {
	return strings.ToLower(strings.TrimSpace(email))
}
