package httpapi

import (
	"net/http"
	"time"

	orgapp "github.com/Romma711/deck-builder/internal/app/org"
	"github.com/Romma711/deck-builder/internal/domain"
	"github.com/Romma711/deck-builder/internal/pkg/validation"
	"github.com/Romma711/deck-builder/internal/ports"
	"github.com/gin-gonic/gin"
)

type orgHandler struct {
	svc     *orgapp.Service
	members ports.MembershipRepository
	tokens  ports.TokenService
}

// RegisterOrgRoutes monta /orgs de la API v3 (Fase 5 SaaS):
//
//	POST   /orgs                          (Auth) crea org, actor = owner
//	GET    /orgs                          (Auth) mis orgs con mi rol
//	GET    /orgs/:orgid                   (miembro)
//	PATCH  /orgs/:orgid                   (owner) renombrar
//	DELETE /orgs/:orgid                   (owner)
//	GET    /orgs/:orgid/members           (miembro)
//	POST   /orgs/:orgid/members           (admin; owner para roles altos)
//	PATCH  /orgs/:orgid/members/:userid   (owner) cambiar rol
//	DELETE /orgs/:orgid/members/:userid   (admin+ o auto-salida)
//
// El middleware RBAC pone la barrera coarse (pertenencia/rol mínimo); las
// reglas finas de jerarquía las aplica el caso de uso.
func RegisterOrgRoutes(rg *gin.RouterGroup, svc *orgapp.Service, memberships ports.MembershipRepository, tokens ports.TokenService) {
	h := &orgHandler{svc: svc, members: memberships, tokens: tokens}

	rg.POST("/orgs", Auth(tokens), h.create)
	rg.GET("/orgs", Auth(tokens), h.listMine)

	rg.GET("/orgs/:orgid", Auth(tokens), RequireOrgRole(memberships, domain.RoleMember), h.get)
	rg.PATCH("/orgs/:orgid", Auth(tokens), RequireOrgRole(memberships, domain.RoleOwner), h.update)
	rg.DELETE("/orgs/:orgid", Auth(tokens), RequireOrgRole(memberships, domain.RoleOwner), h.delete)

	rg.GET("/orgs/:orgid/members", Auth(tokens), RequireOrgRole(memberships, domain.RoleMember), h.listMembers)
	rg.POST("/orgs/:orgid/members", Auth(tokens), RequireOrgRole(memberships, domain.RoleAdmin), h.addMember)
	rg.PATCH("/orgs/:orgid/members/:userid", Auth(tokens), RequireOrgRole(memberships, domain.RoleAdmin), h.changeRole)
	rg.DELETE("/orgs/:orgid/members/:userid", Auth(tokens), RequireOrgRole(memberships, domain.RoleMember), h.removeMember)

	// Invitaciones por link: gestión admin+, vista previa pública (el token
	// es el secreto) y aceptación con sesión.
	rg.POST("/orgs/:orgid/invitations", Auth(tokens), RequireOrgRole(memberships, domain.RoleAdmin), h.createInvitation)
	rg.GET("/orgs/:orgid/invitations", Auth(tokens), RequireOrgRole(memberships, domain.RoleAdmin), h.listInvitations)
	rg.DELETE("/orgs/:orgid/invitations/:invid", Auth(tokens), RequireOrgRole(memberships, domain.RoleAdmin), h.revokeInvitation)
	rg.GET("/invitations/:token", h.previewInvitation)
	rg.POST("/invitations/:token/accept", Auth(tokens), h.acceptInvitation)
}

// --- request/response ---

type orgRequest struct {
	Name string `json:"name" validate:"required,min=2,max=100"`
	Slug string `json:"slug" validate:"omitempty,min=2,max=50"`
}

type orgResponse struct {
	ID        string      `json:"id"`
	Slug      string      `json:"slug"`
	Name      string      `json:"name"`
	Role      domain.Role `json:"role,omitempty"`
	CreatedAt time.Time   `json:"created_at"`
	UpdatedAt time.Time   `json:"updated_at"`
}

func newOrgResponse(org *domain.Organization) orgResponse {
	return orgResponse{
		ID: org.ID, Slug: org.Slug, Name: org.Name, Role: org.Role,
		CreatedAt: org.CreatedAt, UpdatedAt: org.UpdatedAt,
	}
}

type memberAddRequest struct {
	User string      `json:"user" validate:"required,min=1,max=30"` // username
	Role domain.Role `json:"role" validate:"omitempty,oneof=admin member owner"`
}

type memberRoleRequest struct {
	Role domain.Role `json:"role" validate:"required,oneof=admin member owner"`
}

type memberResponse struct {
	UserID   string      `json:"user_id"`
	Username string      `json:"username"`
	Email    string      `json:"email,omitempty"` // vacío si el que consulta no es admin+
	Role     domain.Role `json:"role"`
	JoinedAt time.Time   `json:"joined_at"`
}

// --- handlers ---

func (h *orgHandler) create(c *gin.Context) {
	var req orgRequest
	if !bind(c, &req) {
		return
	}
	user, ok := CurrentUser(c)
	if !ok {
		RespondError(c, domain.ErrUnauthorized)
		return
	}

	org, err := h.svc.Create(c.Request.Context(), user.ID, orgapp.CreateInput{
		Name: req.Name, Slug: req.Slug,
	})
	if err != nil {
		RespondError(c, err)
		return
	}
	c.JSON(http.StatusCreated, newOrgResponse(org))
}

func (h *orgHandler) listMine(c *gin.Context) {
	user, ok := CurrentUser(c)
	if !ok {
		RespondError(c, domain.ErrUnauthorized)
		return
	}
	orgs, err := h.svc.ListMine(c.Request.Context(), user.ID)
	if err != nil {
		RespondError(c, err)
		return
	}
	items := make([]orgResponse, 0, len(orgs))
	for i := range orgs {
		items = append(items, newOrgResponse(&orgs[i]))
	}
	c.JSON(http.StatusOK, gin.H{"data": items, "meta": gin.H{"total": len(items)}})
}

func (h *orgHandler) get(c *gin.Context) {
	user, _ := CurrentUser(c)
	org, err := h.svc.Get(c.Request.Context(), c.Param("orgid"), user.ID)
	if err != nil {
		RespondError(c, err)
		return
	}
	c.JSON(http.StatusOK, newOrgResponse(org))
}

func (h *orgHandler) update(c *gin.Context) {
	var req orgRequest
	if !bind(c, &req) {
		return
	}
	user, _ := CurrentUser(c)
	org, err := h.svc.Update(c.Request.Context(), c.Param("orgid"), user.ID, orgapp.UpdateInput{
		Name: req.Name, Slug: req.Slug,
	})
	if err != nil {
		RespondError(c, err)
		return
	}
	c.JSON(http.StatusOK, newOrgResponse(org))
}

func (h *orgHandler) delete(c *gin.Context) {
	user, _ := CurrentUser(c)
	if err := h.svc.Delete(c.Request.Context(), c.Param("orgid"), user.ID); err != nil {
		RespondError(c, err)
		return
	}
	c.Status(http.StatusNoContent)
}

func (h *orgHandler) listMembers(c *gin.Context) {
	user, _ := CurrentUser(c)
	members, err := h.svc.ListMembers(c.Request.Context(), c.Param("orgid"), user.ID)
	if err != nil {
		RespondError(c, err)
		return
	}
	items := make([]memberResponse, 0, len(members))
	for _, m := range members {
		items = append(items, memberResponse{
			UserID: m.UserID, Username: m.Username, Email: m.Email,
			Role: m.Role, JoinedAt: m.JoinedAt,
		})
	}
	c.JSON(http.StatusOK, gin.H{"data": items, "meta": gin.H{"total": len(items)}})
}

func (h *orgHandler) addMember(c *gin.Context) {
	var req memberAddRequest
	if !bind(c, &req) {
		return
	}
	user, _ := CurrentUser(c)

	member, err := h.svc.AddMember(c.Request.Context(), c.Param("orgid"), user.ID, req.User, req.Role)
	if err != nil {
		RespondError(c, err)
		return
	}
	c.JSON(http.StatusCreated, memberResponse{
		UserID: member.UserID, Username: member.Username, Email: member.Email,
		Role: member.Role, JoinedAt: member.JoinedAt,
	})
}

func (h *orgHandler) changeRole(c *gin.Context) {
	var req memberRoleRequest
	if !bind(c, &req) {
		return
	}
	user, _ := CurrentUser(c)

	if err := h.svc.ChangeRole(c.Request.Context(), c.Param("orgid"), user.ID, c.Param("userid"), req.Role); err != nil {
		RespondError(c, err)
		return
	}
	c.Status(http.StatusNoContent)
}

func (h *orgHandler) removeMember(c *gin.Context) {
	user, _ := CurrentUser(c)
	if err := h.svc.RemoveMember(c.Request.Context(), c.Param("orgid"), user.ID, c.Param("userid")); err != nil {
		RespondError(c, err)
		return
	}
	c.Status(http.StatusNoContent)
}

// bind combina el parseo del JSON con la validación de etiquetas y responde
// el error correspondiente (400 con detalle).
func bind(c *gin.Context, dst any) bool {
	if err := c.ShouldBindJSON(dst); err != nil {
		RespondError(c, &domain.ValidationError{Message: "JSON inválido"})
		return false
	}
	if err := validation.Struct(dst); err != nil {
		RespondError(c, err)
		return false
	}
	return true
}

// --- invitaciones ---

type invitationRequest struct {
	Role domain.Role `json:"role" validate:"omitempty,oneof=admin member owner"`
}

type invitationResponse struct {
	ID          string      `json:"id"`
	Role        domain.Role `json:"role"`
	ExpiresAt   time.Time   `json:"expires_at"`
	CreatedAt   time.Time   `json:"created_at"`
	InviterName string      `json:"inviter_username,omitempty"`
	// Token solo viaja en la respuesta de creación (no se puede recuperar).
	Token string `json:"token,omitempty"`
}

type invitationPreviewResponse struct {
	OrgID       string      `json:"org_id"`
	OrgName     string      `json:"org_name"`
	OrgSlug     string      `json:"org_slug"`
	Role        domain.Role `json:"role"`
	InviterName string      `json:"inviter_username,omitempty"`
	ExpiresAt   time.Time   `json:"expires_at"`
}

func newInvitationResponse(inv *domain.Invitation) invitationResponse {
	return invitationResponse{
		ID: inv.ID, Role: inv.Role, ExpiresAt: inv.ExpiresAt,
		CreatedAt: inv.CreatedAt, InviterName: inv.InviterName,
	}
}

func (h *orgHandler) createInvitation(c *gin.Context) {
	var req invitationRequest
	if !bind(c, &req) {
		return
	}
	user, _ := CurrentUser(c)
	inv, token, err := h.svc.CreateInvitation(c.Request.Context(), c.Param("orgid"), user.ID, req.Role)
	if err != nil {
		RespondError(c, err)
		return
	}
	resp := newInvitationResponse(inv)
	resp.Token = token
	c.JSON(http.StatusCreated, resp)
}

func (h *orgHandler) listInvitations(c *gin.Context) {
	user, _ := CurrentUser(c)
	invs, err := h.svc.ListInvitations(c.Request.Context(), c.Param("orgid"), user.ID)
	if err != nil {
		RespondError(c, err)
		return
	}
	items := make([]invitationResponse, 0, len(invs))
	for i := range invs {
		items = append(items, newInvitationResponse(&invs[i]))
	}
	c.JSON(http.StatusOK, gin.H{"data": items, "meta": gin.H{"total": len(items)}})
}

func (h *orgHandler) revokeInvitation(c *gin.Context) {
	user, _ := CurrentUser(c)
	if err := h.svc.RevokeInvitation(c.Request.Context(), c.Param("orgid"), user.ID, c.Param("invid")); err != nil {
		RespondError(c, err)
		return
	}
	c.Status(http.StatusNoContent)
}

func (h *orgHandler) previewInvitation(c *gin.Context) {
	inv, err := h.svc.PreviewInvitation(c.Request.Context(), c.Param("token"))
	if err != nil {
		RespondError(c, err)
		return
	}
	c.JSON(http.StatusOK, invitationPreviewResponse{
		OrgID: inv.OrgID, OrgName: inv.OrgName, OrgSlug: inv.OrgSlug,
		Role: inv.Role, InviterName: inv.InviterName, ExpiresAt: inv.ExpiresAt,
	})
}

func (h *orgHandler) acceptInvitation(c *gin.Context) {
	user, _ := CurrentUser(c)
	org, err := h.svc.AcceptInvitation(c.Request.Context(), c.Param("token"), user.ID)
	if err != nil {
		RespondError(c, err)
		return
	}
	c.JSON(http.StatusOK, newOrgResponse(org))
}
