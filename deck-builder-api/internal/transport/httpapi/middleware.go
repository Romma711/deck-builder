package httpapi

import (
	"errors"
	"fmt"
	"strings"

	"github.com/Romma711/deck-builder/internal/domain"
	"github.com/Romma711/deck-builder/internal/ports"
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

// fmtForbidden envuelve ErrForbidden con un mensaje para el cliente
// (el transporte responde 403 con code/message, sin detalle interno).
func fmtForbidden(msg string) error {
	return fmt.Errorf("%w: %s", domain.ErrForbidden, msg)
}

// ContextUserKey es la clave de gin.Context donde queda el usuario autenticado.
const ContextUserKey = "auth.user"

// AuthUser es la identidad verificada del request (inyectada por Auth).
type AuthUser struct {
	ID       string
	Username string
}

// Auth es el middleware que verifica el Bearer token UNA sola vez por request
// e inyecta la identidad en el contexto (los handlers no re-parsean el token).
// Además adjunta el user ID al context.Context del request: es lo que las
// políticas RLS de PostgreSQL usan para saber quién consulta.
//
// PUNTO DE EXTENSIÓN (futuro e-commerce): las API keys de tienda/parners se
// conectarán aquí como un middleware alternativo (misma firma gin.HandlerFunc)
// que resuelva la key → tenant y llame a setRequestUser con la identidad
// resultante; el resto de la capa de transporte no cambia.
func Auth(tokens ports.TokenService) gin.HandlerFunc {
	return func(c *gin.Context) {
		claims, err := parseBearer(c, tokens)
		if err != nil {
			RespondError(c, err)
			return
		}
		setRequestUser(c, claims)
		c.Next()
	}
}

// AuthOptional es como Auth pero tolera la ausencia de token (lecturas
// públicas que pueden beneficiarse de identidad: el dueño ve sus mazos
// privados en GET /decks sin cambiar de endpoint). Si el header EXISTE y es
// inválido → 401 (no se ignora un token malicioso en silencio).
func AuthOptional(tokens ports.TokenService) gin.HandlerFunc {
	return func(c *gin.Context) {
		header := c.GetHeader("Authorization")
		if strings.TrimSpace(header) == "" {
			c.Next() // anónimo
			return
		}
		claims, err := parseBearer(c, tokens)
		if err != nil {
			RespondError(c, err)
			return
		}
		setRequestUser(c, claims)
		c.Next()
	}
}

// RequireOrgRole es el middleware RBAC coarse: exige que el usuario sea
// miembro de la org del path con al menos el rol mínimo indicado. Resuelve el
// rol en la BASE por request (rol siempre fresco; ver decisión Fase 5).
// No miembro / rol insuficiente → 403. org_id malformado → 400.
//
// Los casos de negocio finos (jerarquía, último owner...) vive en el caso de
// uso (internal/app/org): este middleware solo evita llegar ahí sin pertenencia.
func RequireOrgRole(memberships ports.MembershipRepository, min domain.Role) gin.HandlerFunc {
	return func(c *gin.Context) {
		user, ok := CurrentUser(c)
		if !ok {
			RespondError(c, domain.ErrUnauthorized)
			return
		}
		orgID := c.Param("orgid")
		if _, err := uuid.Parse(orgID); err != nil {
			RespondError(c, domain.ErrInvalidID)
			return
		}

		m, err := memberships.FindByOrgAndUser(c.Request.Context(), orgID, user.ID)
		if err != nil {
			if errors.Is(err, domain.ErrNotFound) {
				// No revelamos si la org existe: "no tienes acceso" basta.
				RespondError(c, fmtForbidden("no eres miembro de la organización"))
				return
			}
			if errors.Is(err, domain.ErrInvalidID) {
				RespondError(c, err)
				return
			}
			RespondError(c, err)
			return
		}
		if !m.Role.AtLeast(min) {
			RespondError(c, fmtForbidden("se requiere rol "+string(min)+" (tienes "+string(m.Role)+")"))
			return
		}
		c.Next()
	}
}

// --- helpers ---

func parseBearer(c *gin.Context, tokens ports.TokenService) (*ports.AccessTokenClaims, error) {
	header := c.GetHeader("Authorization")
	if !strings.HasPrefix(header, "Bearer ") {
		return nil, domain.ErrUnauthorized
	}
	raw := strings.TrimSpace(strings.TrimPrefix(header, "Bearer "))
	if raw == "" {
		return nil, domain.ErrUnauthorized
	}

	claims, err := tokens.ParseAccessToken(raw)
	if err != nil {
		// Cualquier error de parseo (firma, algoritmo, expiración, claims)
		// cae aquí como 401: nunca se filtra el motivo.
		return nil, err
	}
	return claims, nil
}

func setRequestUser(c *gin.Context, claims *ports.AccessTokenClaims) {
	c.Set(ContextUserKey, AuthUser{ID: claims.UserID, Username: claims.Username})
	// Contexto del request → lo lee DeckRepository para las políticas RLS.
	c.Request = c.Request.WithContext(
		domain.WithUserID(c.Request.Context(), claims.UserID))
}

// CurrentUser recupera la identidad inyectada por Auth.
func CurrentUser(c *gin.Context) (AuthUser, bool) {
	value, exists := c.Get(ContextUserKey)
	if !exists {
		return AuthUser{}, false
	}
	user, ok := value.(AuthUser)
	return user, ok
}
