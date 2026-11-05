package httpapi

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/Romma711/deck-builder/internal/domain"
	authinfra "github.com/Romma711/deck-builder/internal/infra/auth"
	"github.com/Romma711/deck-builder/internal/ports"
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

// newSecureEngine arma un engine con el middleware Auth usando el servicio
// JWT real (mismas reglas de algoritmo/firma que producción).
func newSecureEngine(t *testing.T, handler gin.HandlerFunc) (*gin.Engine, string) {
	t.Helper()
	gin.SetMode(gin.TestMode)

	tokens := authinfra.NewTokenService("clave-de-prueba", 15*time.Minute)
	token, err := tokens.IssueAccessToken(&domain.User{ID: "user-1", Username: "romina"})
	if err != nil {
		t.Fatalf("no pude emitir el token: %v", err)
	}

	engine := gin.New()
	engine.GET("/privado", Auth(tokens), handler)
	return engine, token
}

func TestAuthMiddleware_TokenValido_InyectaUsuario(t *testing.T) {
	var gotUser AuthUser
	handler := func(c *gin.Context) {
		user, ok := CurrentUser(c)
		if !ok {
			c.Status(http.StatusInternalServerError)
			return
		}
		gotUser = user
		c.Status(http.StatusOK)
	}

	engine, token := newSecureEngine(t, handler)

	req := httptest.NewRequest(http.MethodGet, "/privado", nil)
	req.Header.Set("Authorization", "Bearer "+token)
	w := httptest.NewRecorder()
	engine.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("status = %d, esperaba 200 (body: %s)", w.Code, w.Body.String())
	}
	if gotUser.ID != "user-1" || gotUser.Username != "romina" {
		t.Errorf("usuario inyectado = %+v, esperaba user-1/romina", gotUser)
	}
}

func TestAuthMiddleware_SinHeader_401(t *testing.T) {
	engine, _ := newSecureEngine(t, okHandler)

	req := httptest.NewRequest(http.MethodGet, "/privado", nil)
	w := httptest.NewRecorder()
	engine.ServeHTTP(w, req)

	if w.Code != http.StatusUnauthorized {
		t.Fatalf("status = %d, esperaba 401", w.Code)
	}
}

func TestAuthMiddleware_SoloPrefixBearer_401(t *testing.T) {
	engine, token := newSecureEngine(t, okHandler)

	req := httptest.NewRequest(http.MethodGet, "/privado", nil)
	req.Header.Set("Authorization", token) // sin "Bearer "
	w := httptest.NewRecorder()
	engine.ServeHTTP(w, req)

	if w.Code != http.StatusUnauthorized {
		t.Fatalf("status = %d, esperaba 401", w.Code)
	}
}

func TestAuthMiddleware_TokenInvalido_401(t *testing.T) {
	engine, _ := newSecureEngine(t, okHandler)

	req := httptest.NewRequest(http.MethodGet, "/privado", nil)
	req.Header.Set("Authorization", "Bearer token-falsificado")
	w := httptest.NewRecorder()
	engine.ServeHTTP(w, req)

	if w.Code != http.StatusUnauthorized {
		t.Fatalf("status = %d, esperaba 401", w.Code)
	}
}

func TestAuthMiddleware_TokenFirmadoConOtraClave_401(t *testing.T) {
	engine, _ := newSecureEngine(t, okHandler)

	// Token bien formado pero firmado con otra clave.
	attacker := authinfra.NewTokenService("clave-del-atacante", 15*time.Minute)
	malicious, _ := attacker.IssueAccessToken(&domain.User{ID: "user-1", Username: "romina"})

	req := httptest.NewRequest(http.MethodGet, "/privado", nil)
	req.Header.Set("Authorization", "Bearer "+malicious)
	w := httptest.NewRecorder()
	engine.ServeHTTP(w, req)

	if w.Code != http.StatusUnauthorized {
		t.Fatalf("status = %d, esperaba 401", w.Code)
	}
}

func okHandler(c *gin.Context) { c.Status(http.StatusOK) }

// --- AuthOptional ---

func newOptionalEngine(t *testing.T, handler gin.HandlerFunc) (*gin.Engine, string) {
	t.Helper()
	gin.SetMode(gin.TestMode)

	tokens := authinfra.NewTokenService("clave-de-prueba", 15*time.Minute)
	token, err := tokens.IssueAccessToken(&domain.User{ID: "user-1", Username: "romina"})
	if err != nil {
		t.Fatalf("no pude emitir el token: %v", err)
	}

	engine := gin.New()
	engine.GET("/publico", AuthOptional(tokens), handler)
	return engine, token
}

func TestAuthOptional_SinHeader_AnonimoContinua(t *testing.T) {
	called := false
	handler := func(c *gin.Context) {
		called = true
		if _, ok := CurrentUser(c); ok {
			c.Status(http.StatusInternalServerError) // no debe haber usuario
			return
		}
		c.Status(http.StatusOK)
	}
	engine, _ := newOptionalEngine(t, handler)

	req := httptest.NewRequest(http.MethodGet, "/publico", nil)
	w := httptest.NewRecorder()
	engine.ServeHTTP(w, req)

	if w.Code != http.StatusOK || !called {
		t.Fatalf("status = %d, called = %v; esperaba 200 y handler ejecutado", w.Code, called)
	}
}

func TestAuthOptional_ConToken_IdentityYContextoRLS(t *testing.T) {
	var ctxUser string
	handler := func(c *gin.Context) {
		ctxUser = domain.UserIDFromContext(c.Request.Context())
		c.Status(http.StatusOK)
	}
	engine, token := newOptionalEngine(t, handler)

	req := httptest.NewRequest(http.MethodGet, "/publico", nil)
	req.Header.Set("Authorization", "Bearer "+token)
	w := httptest.NewRecorder()
	engine.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("status = %d, esperaba 200", w.Code)
	}
	if ctxUser != "user-1" {
		// Sin esto, RLS no sabría quién consulta y el dueño no vería sus
		// mazos privados en las lecturas públicas.
		t.Errorf("domain.UserIDFromContext = %q, esperaba user-1", ctxUser)
	}
}

func TestAuthOptional_TokenInvalido_401(t *testing.T) {
	engine, _ := newOptionalEngine(t, okHandler)

	req := httptest.NewRequest(http.MethodGet, "/publico", nil)
	req.Header.Set("Authorization", "Bearer token-falsificado")
	w := httptest.NewRecorder()
	engine.ServeHTTP(w, req)

	if w.Code != http.StatusUnauthorized {
		t.Fatalf("status = %d, esperaba 401 (header presente pero inválido)", w.Code)
	}
}

// --- RequireOrgRole (RBAC con lookup en DB) ---

// rbacMembersFake simula la consulta de membership con el comportamiento del
// repo real (ErrInvalidID con org_id basura, ErrNotFound si no es miembro).
type rbacMembersFake struct {
	member bool
	role   domain.Role
}

func (f rbacMembersFake) FindByOrgAndUser(_ context.Context, orgID, _ string) (*domain.Membership, error) {
	if _, err := uuid.Parse(orgID); err != nil {
		return nil, domain.ErrInvalidID
	}
	if !f.member {
		return nil, domain.ErrNotFound
	}
	return &domain.Membership{OrgID: orgID, UserID: "user-1", Role: f.role}, nil
}

func (f rbacMembersFake) Add(_ context.Context, _ *domain.Membership) error              { return nil }
func (f rbacMembersFake) UpdateRole(_ context.Context, _, _ string, _ domain.Role) error { return nil }
func (f rbacMembersFake) Remove(_ context.Context, _, _ string) error                    { return nil }
func (f rbacMembersFake) CountOwners(_ context.Context, _ string) (int, error)           { return 1, nil }
func (f rbacMembersFake) ListMembersByOrg(_ context.Context, _ string) ([]ports.MemberInfo, error) {
	return nil, nil
}

const orgIDValid = "10000000-0000-7000-8000-00000000000f"

func newRBACEngine(t *testing.T, members rbacMembersFake, min domain.Role) (*gin.Engine, string) {
	t.Helper()
	gin.SetMode(gin.TestMode)

	tokens := authinfra.NewTokenService("clave-de-prueba", 15*time.Minute)
	token, err := tokens.IssueAccessToken(&domain.User{ID: "user-1", Username: "romina"})
	if err != nil {
		t.Fatalf("no pude emitir el token: %v", err)
	}

	engine := gin.New()
	engine.GET("/orgs/:orgid", Auth(tokens), RequireOrgRole(members, min), okHandler)
	return engine, token
}

func rbacRequest(engine *gin.Engine, token, orgID string) *httptest.ResponseRecorder {
	req := httptest.NewRequest(http.MethodGet, "/orgs/"+orgID, nil)
	if token != "" {
		req.Header.Set("Authorization", "Bearer "+token)
	}
	w := httptest.NewRecorder()
	engine.ServeHTTP(w, req)
	return w
}

func TestRequireOrgRole_Anonimo_401(t *testing.T) {
	engine, _ := newRBACEngine(t, rbacMembersFake{member: true, role: domain.RoleOwner}, domain.RoleMember)
	if w := rbacRequest(engine, "", orgIDValid); w.Code != http.StatusUnauthorized {
		t.Fatalf("status = %d, esperaba 401", w.Code)
	}
}

func TestRequireOrgRole_OrgIdBasura_400(t *testing.T) {
	engine, token := newRBACEngine(t, rbacMembersFake{member: true, role: domain.RoleOwner}, domain.RoleMember)
	if w := rbacRequest(engine, token, "basura"); w.Code != http.StatusBadRequest {
		t.Fatalf("status = %d, esperaba 400", w.Code)
	}
}

func TestRequireOrgRole_NoMiembro_403(t *testing.T) {
	engine, token := newRBACEngine(t, rbacMembersFake{member: false}, domain.RoleMember)
	if w := rbacRequest(engine, token, orgIDValid); w.Code != http.StatusForbidden {
		t.Fatalf("status = %d, esperaba 403", w.Code)
	}
}

func TestRequireOrgRole_RolInsuficiente_403(t *testing.T) {
	// Se pide admin y el usuario es member.
	engine, token := newRBACEngine(t, rbacMembersFake{member: true, role: domain.RoleMember}, domain.RoleAdmin)
	if w := rbacRequest(engine, token, orgIDValid); w.Code != http.StatusForbidden {
		t.Fatalf("status = %d, esperaba 403 (rol insuficiente)", w.Code)
	}
}

func TestRequireOrgRole_RolSuficiente_200(t *testing.T) {
	// Se pide admin y el usuario es owner.
	engine, token := newRBACEngine(t, rbacMembersFake{member: true, role: domain.RoleOwner}, domain.RoleAdmin)
	if w := rbacRequest(engine, token, orgIDValid); w.Code != http.StatusOK {
		t.Fatalf("status = %d, esperaba 200 (body: %s)", w.Code, w.Body.String())
	}
}
