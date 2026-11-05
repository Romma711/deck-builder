package auth

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/Romma711/deck-builder/internal/domain"
	"github.com/Romma711/deck-builder/internal/ports"
)

// --- Fakes (implementan los puertos en memoria, sin infraestructura) ---

type fakeUserRepo struct {
	mu     sync.Mutex
	nextID int
	users  map[string]*domain.User // id → user
}

func newFakeUserRepo() *fakeUserRepo {
	return &fakeUserRepo{users: map[string]*domain.User{}}
}

func (f *fakeUserRepo) Create(_ context.Context, user *domain.User) error {
	f.mu.Lock()
	defer f.mu.Unlock()
	for _, existing := range f.users {
		if existing.Username == user.Username || existing.Email == user.Email {
			return fmt.Errorf("%w: duplicado", domain.ErrConflict)
		}
	}
	f.nextID++
	user.ID = fmt.Sprintf("user-%d", f.nextID)
	f.users[user.ID] = user
	return nil
}

func (f *fakeUserRepo) FindByUsername(_ context.Context, username string) (*domain.User, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	for _, u := range f.users {
		if u.Username == username {
			copy := *u
			return &copy, nil
		}
	}
	return nil, domain.ErrNotFound
}

func (f *fakeUserRepo) FindByID(_ context.Context, id string) (*domain.User, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	if u, ok := f.users[id]; ok {
		copy := *u
		return &copy, nil
	}
	return nil, domain.ErrNotFound
}

func (f *fakeUserRepo) Update(_ context.Context, user *domain.User) error {
	f.mu.Lock()
	defer f.mu.Unlock()
	if _, ok := f.users[user.ID]; !ok {
		return domain.ErrNotFound
	}
	f.users[user.ID] = user
	return nil
}

func (f *fakeUserRepo) Delete(_ context.Context, id string) error {
	f.mu.Lock()
	defer f.mu.Unlock()
	if _, ok := f.users[id]; !ok {
		return domain.ErrNotFound
	}
	delete(f.users, id)
	return nil
}

type fakeTokenRepo struct {
	mu     sync.Mutex
	nextID int
	tokens map[string]*domain.RefreshToken // id → token
}

func newFakeTokenRepo() *fakeTokenRepo {
	return &fakeTokenRepo{tokens: map[string]*domain.RefreshToken{}}
}

func (f *fakeTokenRepo) Save(_ context.Context, token *domain.RefreshToken) error {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.nextID++
	token.ID = fmt.Sprintf("rt-%d", f.nextID)
	f.tokens[token.ID] = token
	return nil
}

func (f *fakeTokenRepo) FindByHash(_ context.Context, hash string) (*domain.RefreshToken, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	for _, t := range f.tokens {
		if t.TokenHash == hash {
			copy := *t
			return &copy, nil
		}
	}
	return nil, domain.ErrNotFound
}

func (f *fakeTokenRepo) Revoke(_ context.Context, id string) error {
	f.mu.Lock()
	defer f.mu.Unlock()
	if t, ok := f.tokens[id]; ok {
		now := time.Now().UTC()
		t.RevokedAt = &now
	}
	return nil
}

func (f *fakeTokenRepo) RevokeAllForUser(_ context.Context, userID string) error {
	f.mu.Lock()
	defer f.mu.Unlock()
	now := time.Now().UTC()
	for _, t := range f.tokens {
		if t.UserID == userID && t.RevokedAt == nil {
			t.RevokedAt = &now
		}
	}
	return nil
}

// fakeTokenSvc emite tokens "access-<userID>" y los parsea de vuelta.
type fakeTokenSvc struct{}

func (fakeTokenSvc) IssueAccessToken(user *domain.User) (string, error) {
	return "access-" + user.ID, nil
}

func (fakeTokenSvc) ParseAccessToken(token string) (*ports.AccessTokenClaims, error) {
	if !strings.HasPrefix(token, "access-") {
		return nil, fmt.Errorf("%w: token inválido", domain.ErrUnauthorized)
	}
	return &ports.AccessTokenClaims{UserID: strings.TrimPrefix(token, "access-")}, nil
}

// fakeHasher: hash trivial "sha-ish" para tests rápidos (NO usar en producción).
type fakeHasher struct{}

func (fakeHasher) Hash(plain string) (string, error) { return "hash:" + plain, nil }
func (fakeHasher) Compare(hash, plain string) bool   { return hash == "hash:"+plain }

// --- Harness ---

func newTestService() (*Service, *fakeUserRepo, *fakeTokenRepo) {
	users := newFakeUserRepo()
	refresh := newFakeTokenRepo()
	svc := NewService(users, refresh, fakeTokenSvc{}, fakeHasher{}, 15*time.Minute, 7*24*time.Hour)
	return svc, users, refresh
}

func registerUser(t *testing.T, svc *Service) *domain.User {
	t.Helper()
	user, err := svc.Register(context.Background(), RegisterInput{
		Username: "romina",
		Email:    "romina@example.com",
		Password: "supersecreta123",
	})
	if err != nil {
		t.Fatalf("Register falló: %v", err)
	}
	return user
}

// --- Tests ---

func TestRegister_Success_HasheaLaContraseña(t *testing.T) {
	svc, users, _ := newTestService()
	registerUser(t, svc)

	// El hash guardado NUNCA debe ser la contraseña en claro.
	stored := users.users["user-1"]
	if stored.PasswordHash == "supersecreta123" {
		t.Fatal("la contraseña se guardó en claro")
	}
	if !strings.HasPrefix(stored.PasswordHash, "hash:") {
		t.Errorf("se esperaba el hash del hasher, obtuve %q", stored.PasswordHash)
	}
}

func TestRegister_PasswordCorta_ValidationError(t *testing.T) {
	svc, _, _ := newTestService()
	_, err := svc.Register(context.Background(), RegisterInput{
		Username: "romina", Email: "r@example.com", Password: "corta",
	})
	var ve *domain.ValidationError
	if !errors.As(err, &ve) {
		t.Fatalf("esperaba *domain.ValidationError, obtuve %v", err)
	}
	if ve.Details["Password"] == "" {
		t.Errorf("falta el detalle del campo Password: %v", ve.Details)
	}
}

func TestRegister_Duplicado_Conflict(t *testing.T) {
	svc, _, _ := newTestService()
	registerUser(t, svc)

	_, err := svc.Register(context.Background(), RegisterInput{
		Username: "romina", Email: "otra@example.com", Password: "supersecreta123",
	})
	if !errors.Is(err, domain.ErrConflict) {
		t.Fatalf("esperaba ErrConflict, obtuve %v", err)
	}
}

func TestLogin_UsuarioDesconocidoYContraseñaIncorrecta_MismoError(t *testing.T) {
	svc, _, _ := newTestService()
	registerUser(t, svc)

	_, errUnknown := svc.Login(context.Background(), "nadie", "supersecreta123")
	_, errBadPass := svc.Login(context.Background(), "romina", "otracosa123")

	// Sin enumeración de usuarios: mismo error genérico en ambos casos.
	if !errors.Is(errUnknown, domain.ErrUnauthorized) || !errors.Is(errBadPass, domain.ErrUnauthorized) {
		t.Fatalf("esperaba ErrUnauthorized en ambos, obtuve %v / %v", errUnknown, errBadPass)
	}
	if errUnknown.Error() != errBadPass.Error() {
		t.Errorf("los mensajes difieren y permiten enumerar usuarios: %q vs %q",
			errUnknown.Error(), errBadPass.Error())
	}
}

func TestLogin_Success(t *testing.T) {
	svc, _, refreshRepo := newTestService()
	registerUser(t, svc)

	pair, err := svc.Login(context.Background(), "romina", "supersecreta123")
	if err != nil {
		t.Fatalf("Login falló: %v", err)
	}
	if pair.AccessToken == "" || pair.RefreshToken == "" {
		t.Fatal("el par de tokens viene incompleto")
	}
	if pair.ExpiresIn != int64((15 * time.Minute).Seconds()) {
		t.Errorf("ExpiresIn = %d, esperaba 900", pair.ExpiresIn)
	}

	// El refresh token en claro NO debe guardarse: solo su hash.
	for _, stored := range refreshRepo.tokens {
		if stored.TokenHash == pair.RefreshToken {
			t.Fatal("el refresh token se guardó en claro")
		}
	}
}

func TestRefresh_Rotacion(t *testing.T) {
	svc, _, _ := newTestService()
	registerUser(t, svc)

	first, _ := svc.Login(context.Background(), "romina", "supersecreta123")

	second, err := svc.Refresh(context.Background(), first.RefreshToken)
	if err != nil {
		t.Fatalf("Refresh falló: %v", err)
	}
	if second.RefreshToken == first.RefreshToken {
		t.Fatal("el refresh token no rotó")
	}

	// Reuso del token viejo: rechazado Y revoca la familia completa (antirobo).
	_, err = svc.Refresh(context.Background(), first.RefreshToken)
	if !errors.Is(err, domain.ErrUnauthorized) {
		t.Fatalf("esperaba ErrUnauthorized con token reusado, obtuve %v", err)
	}

	// El token nuevo también quedó revocado por la familia.
	_, err = svc.Refresh(context.Background(), second.RefreshToken)
	if !errors.Is(err, domain.ErrUnauthorized) {
		t.Fatalf("la familia de tokens debió revocarse, obtuve %v", err)
	}
}

func TestRefresh_TokenDesconocido(t *testing.T) {
	svc, _, _ := newTestService()
	_, err := svc.Refresh(context.Background(), "token-inexistente")
	if !errors.Is(err, domain.ErrUnauthorized) {
		t.Fatalf("esperaba ErrUnauthorized, obtuve %v", err)
	}
}

func TestLogout_Idempotente(t *testing.T) {
	svc, _, _ := newTestService()
	registerUser(t, svc)
	pair, _ := svc.Login(context.Background(), "romina", "supersecreta123")

	if err := svc.Logout(context.Background(), pair.RefreshToken); err != nil {
		t.Fatalf("Logout falló: %v", err)
	}
	// Segundo logout con el mismo token: no debe fallar.
	if err := svc.Logout(context.Background(), pair.RefreshToken); err != nil {
		t.Fatalf("Logout no es idempotente: %v", err)
	}

	// El token ya revocado no sirve para refrescar.
	_, err := svc.Refresh(context.Background(), pair.RefreshToken)
	if !errors.Is(err, domain.ErrUnauthorized) {
		t.Fatalf("token revocado debió ser rechazado, obtuve %v", err)
	}
}

func TestDeleteAccount_RevocaTokens(t *testing.T) {
	svc, users, _ := newTestService()
	user := registerUser(t, svc)
	pair, _ := svc.Login(context.Background(), "romina", "supersecreta123")

	if err := svc.DeleteAccount(context.Background(), user.ID); err != nil {
		t.Fatalf("DeleteAccount falló: %v", err)
	}
	if _, ok := users.users[user.ID]; ok {
		t.Fatal("el usuario no fue eliminado")
	}

	_, err := svc.Refresh(context.Background(), pair.RefreshToken)
	if !errors.Is(err, domain.ErrUnauthorized) {
		t.Fatalf("los tokens del usuario eliminado debían revocarse, obtuve %v", err)
	}
}
