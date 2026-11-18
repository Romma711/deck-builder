// Package org implementa el caso de uso de organizaciones (Fase 5 SaaS):
// crear gestionar miembros y roles (owner > admin > member), y borrar orgs.
//
// Los permisos se resuelven AQUÍ con lookup en la base (siempre frescos);
// el middleware RBAC del transporte es solo una barrera coarse previa.
// Reglas de jerarquía:
//   - owner: todo (rename, borrar org, promover/degradar, gestionar cualquiera)
//   - admin: agregar/quitar miembros con rol member (nunca tocar admin/owner)
//   - member: leer (org, miembros, mazos de la org) y salirse
//   - guardia: el último owner no puede ser degradado ni salirse (la org no
//     queda sin dueño).
package org

import (
	"context"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"errors"
	"fmt"
	"regexp"
	"strings"
	"time"
	"unicode"

	"github.com/Romma711/deck-builder/internal/domain"
	"github.com/Romma711/deck-builder/internal/ports"
	"golang.org/x/text/unicode/norm"
)

// slugPattern define un slug URL-amigable: minúsculas, dígitos y guiones.
var slugPattern = regexp.MustCompile(`^[a-z0-9]+(-[a-z0-9]+)*$`)

type Service struct {
	orgs    ports.OrganizationRepository
	members ports.MembershipRepository
	users   ports.UserRepository
	invites ports.InvitationRepository
	now     func() time.Time // inyectable para tests
}

func NewService(
	orgs ports.OrganizationRepository,
	members ports.MembershipRepository,
	users ports.UserRepository,
	invites ports.InvitationRepository,
) *Service {
	return &Service{
		orgs:    orgs,
		members: members,
		users:   users,
		invites: invites,
		now:     func() time.Time { return time.Now().UTC() },
	}
}

// --- DTOs de entrada ---

type CreateInput struct {
	Name string
	Slug string // opcional: si vacío se deriva del nombre
}

type UpdateInput struct {
	Name string
	Slug string
}

// --- Organización ---

// Create crea la org con el usuario como owner (atómico en el repositorio).
func (s *Service) Create(ctx context.Context, userID string, in CreateInput) (*domain.Organization, error) {
	if userID == "" {
		return nil, domain.ErrUnauthorized
	}

	details := map[string]string{}
	name := strings.TrimSpace(in.Name)
	if len(name) < 2 || len(name) > 100 {
		details["name"] = "min=2,max=100"
	}
	slug := strings.ToLower(strings.TrimSpace(in.Slug))
	if slug == "" {
		slug = slugify(name)
	}
	if !slugPattern.MatchString(slug) || len(slug) > 50 {
		details["slug"] = "lowercase_slug_1_50"
	}
	if len(details) > 0 {
		return nil, &domain.ValidationError{Details: details}
	}

	now := s.now()
	org := &domain.Organization{
		Slug: slug, Name: name,
		CreatedAt: now, UpdatedAt: now,
	}
	// El repositorio inserta org + membership owner en una transacción.
	if err := s.orgs.Create(ctx, org, userID); err != nil {
		return nil, err
	}
	org.Role = domain.RoleOwner
	return org, nil
}

// Get devuelve la org con el rol del consultante (debe ser miembro).
func (s *Service) Get(ctx context.Context, orgID, viewerID string) (*domain.Organization, error) {
	org, _, err := s.requireRole(ctx, orgID, viewerID, domain.RoleMember)
	if err != nil {
		return nil, err
	}
	return org, nil
}

// ListMine lista las orgs del usuario con su rol en cada una.
func (s *Service) ListMine(ctx context.Context, userID string) ([]domain.Organization, error) {
	if userID == "" {
		return nil, domain.ErrUnauthorized
	}
	return s.orgs.ListByUser(ctx, userID)
}

// Update renombra la org (solo owner).
func (s *Service) Update(ctx context.Context, orgID, actorID string, in UpdateInput) (*domain.Organization, error) {
	org, _, err := s.requireRole(ctx, orgID, actorID, domain.RoleOwner)
	if err != nil {
		return nil, err
	}

	details := map[string]string{}
	name := strings.TrimSpace(in.Name)
	if len(name) < 2 || len(name) > 100 {
		details["name"] = "min=2,max=100"
	}
	slug := strings.ToLower(strings.TrimSpace(in.Slug))
	if !slugPattern.MatchString(slug) || len(slug) > 50 {
		details["slug"] = "lowercase_slug_1_50"
	}
	if len(details) > 0 {
		return nil, &domain.ValidationError{Details: details}
	}

	org.Name = name
	org.Slug = slug
	org.UpdatedAt = s.now()
	if err := s.orgs.Update(ctx, org); err != nil {
		return nil, err
	}
	return org, nil
}

// Delete borra la org (solo owner). memberships caen en cascada y los mazos
// de la org quedan personales (FK ON DELETE SET NULL).
func (s *Service) Delete(ctx context.Context, orgID, actorID string) error {
	if _, _, err := s.requireRole(ctx, orgID, actorID, domain.RoleOwner); err != nil {
		return err
	}
	return s.orgs.Delete(ctx, orgID)
}

// --- Miembros ---

// AddMember agrega a un usuario EXISTENTE (por username) con el rol pedido.
// admin solo puede otorgar 'member'; owner cualquiera.
func (s *Service) AddMember(ctx context.Context, orgID, actorID, username string, role domain.Role) (*ports.MemberInfo, error) {
	actor, _, err := s.requireRole(ctx, orgID, actorID, domain.RoleAdmin)
	if err != nil {
		return nil, err
	}

	if role == "" {
		role = domain.RoleMember
	}
	if !domain.ValidRole(role) {
		return nil, &domain.ValidationError{Details: map[string]string{"role": "oneof=admin,member,owner"}}
	}
	if actor.Role != domain.RoleOwner && role != domain.RoleMember {
		return nil, fmt.Errorf("%w: solo un owner puede otorgar roles admin/owner", domain.ErrForbidden)
	}

	target, err := s.users.FindByUsername(ctx, strings.TrimSpace(username))
	if err != nil {
		if errors.Is(err, domain.ErrNotFound) {
			return nil, &domain.ValidationError{
				Details: map[string]string{"user": "unknown"}, Message: "usuario no encontrado: " + username,
			}
		}
		return nil, err
	}

	m := &domain.Membership{OrgID: orgID, UserID: target.ID, Role: role, CreatedAt: s.now()}
	if err := s.members.Add(ctx, m); err != nil {
		return nil, err // ErrConflict si ya es miembro; ErrNotFound si la org no existe
	}

	return &ports.MemberInfo{
		UserID: target.ID, Username: target.Username, Email: target.Email,
		Role: role, JoinedAt: m.CreatedAt,
	}, nil
}

// ChangeRole modifica el rol de un miembro (solo owner).
func (s *Service) ChangeRole(ctx context.Context, orgID, actorID, targetUserID string, newRole domain.Role) error {
	if _, _, err := s.requireRole(ctx, orgID, actorID, domain.RoleOwner); err != nil {
		return err
	}
	if !domain.ValidRole(newRole) {
		return &domain.ValidationError{Details: map[string]string{"role": "oneof=admin,member,owner"}}
	}

	target, err := s.members.FindByOrgAndUser(ctx, orgID, targetUserID)
	if err != nil {
		return err // ErrNotFound: no es miembro
	}
	if target.Role == newRole {
		return nil // idempotente
	}

	// Guardia de "último owner": la org no puede quedarse sin dueño.
	if target.Role == domain.RoleOwner && newRole != domain.RoleOwner {
		owners, err := s.members.CountOwners(ctx, orgID)
		if err != nil {
			return err
		}
		if owners <= 1 {
			return fmt.Errorf("%w: la organización no puede quedarse sin owner", domain.ErrConflict)
		}
	}
	return s.members.UpdateRole(ctx, orgID, targetUserID, newRole)
}

// RemoveMember quita un miembro. Si actor == target es una AUTO-SALIDA
// (permitida a cualquier rol); si no, requiere rol admin+ con las reglas de
// jerarquía (admin solo puede quitar members).
func (s *Service) RemoveMember(ctx context.Context, orgID, actorID, targetUserID string) error {
	target, err := s.members.FindByOrgAndUser(ctx, orgID, targetUserID)
	if err != nil {
		return err // ErrNotFound: no es miembro
	}

	isSelf := actorID == targetUserID

	// Guardia de "último owner" en cualquier vía de salida.
	if target.Role == domain.RoleOwner {
		owners, err := s.members.CountOwners(ctx, orgID)
		if err != nil {
			return err
		}
		if owners <= 1 {
			return fmt.Errorf("%w: el último owner no puede salirse ni ser removido", domain.ErrConflict)
		}
	}

	if isSelf {
		// Auto-salida: cualquier miembro (la guardia de owner ya corrió).
		return s.members.Remove(ctx, orgID, targetUserID)
	}

	actor, _, err := s.requireRole(ctx, orgID, actorID, domain.RoleAdmin)
	if err != nil {
		return err
	}
	if actor.Role != domain.RoleOwner && target.Role != domain.RoleMember {
		return fmt.Errorf("%w: un admin solo puede remover miembros con rol member", domain.ErrForbidden)
	}
	return s.members.Remove(ctx, orgID, targetUserID)
}

// ListMembers lista los miembros (debe ser miembro).
//
// Privacidad: los emails ajenos solo los ven admin y owner (los necesitan
// para gestionar la org). Un member ve el suyo y el username del resto.
func (s *Service) ListMembers(ctx context.Context, orgID, actorID string) ([]ports.MemberInfo, error) {
	_, actor, err := s.requireRole(ctx, orgID, actorID, domain.RoleMember)
	if err != nil {
		return nil, err
	}
	members, err := s.members.ListMembersByOrg(ctx, orgID)
	if err != nil {
		return nil, err
	}
	if !actor.Role.AtLeast(domain.RoleAdmin) {
		for i := range members {
			if members[i].UserID != actorID {
				members[i].Email = ""
			}
		}
	}
	return members, nil
}

// --- Invitaciones por link ---

// CreateInvitation crea un link de invitación (rol mínimo: admin). Mismas
// reglas que AddMember: solo un owner puede invitar como admin u owner.
// Devuelve el token en claro: es la ÚNICA vez que existe (se guarda su hash).
func (s *Service) CreateInvitation(ctx context.Context, orgID, actorID string, role domain.Role) (*domain.Invitation, string, error) {
	_, actor, err := s.requireRole(ctx, orgID, actorID, domain.RoleAdmin)
	if err != nil {
		return nil, "", err
	}
	if role == "" {
		role = domain.RoleMember
	}
	if !domain.ValidRole(role) {
		return nil, "", &domain.ValidationError{Details: map[string]string{"role": "oneof=admin,member,owner"}}
	}
	if actor.Role != domain.RoleOwner && role != domain.RoleMember {
		return nil, "", fmt.Errorf("%w: solo un owner puede invitar como admin/owner", domain.ErrForbidden)
	}

	buf := make([]byte, 32)
	if _, err := rand.Read(buf); err != nil {
		return nil, "", fmt.Errorf("%w: generar token de invitación: %v", domain.ErrInternal, err)
	}
	token := base64.RawURLEncoding.EncodeToString(buf)

	now := s.now()
	inv := &domain.Invitation{
		OrgID: orgID, Role: role, CreatedBy: actorID,
		ExpiresAt: now.Add(domain.InvitationTTL), CreatedAt: now,
	}
	if err := s.invites.Create(ctx, inv, hashInvitationToken(token)); err != nil {
		return nil, "", err
	}
	return inv, token, nil
}

// ListInvitations lista las invitaciones activas (rol mínimo: admin).
func (s *Service) ListInvitations(ctx context.Context, orgID, actorID string) ([]domain.Invitation, error) {
	if _, _, err := s.requireRole(ctx, orgID, actorID, domain.RoleAdmin); err != nil {
		return nil, err
	}
	return s.invites.ListActiveByOrg(ctx, orgID, s.now())
}

// RevokeInvitation revoca una invitación activa (rol mínimo: admin).
func (s *Service) RevokeInvitation(ctx context.Context, orgID, actorID, invitationID string) error {
	if _, _, err := s.requireRole(ctx, orgID, actorID, domain.RoleAdmin); err != nil {
		return err
	}
	return s.invites.Revoke(ctx, orgID, invitationID, s.now())
}

// PreviewInvitation muestra a quién y a qué invita un link activo. No exige
// sesión: el token ya es el secreto. Usado, vencido o revocado → ErrNotFound.
func (s *Service) PreviewInvitation(ctx context.Context, token string) (*domain.Invitation, error) {
	return s.invites.FindActiveByTokenHash(ctx, hashInvitationToken(token), s.now())
}

// AcceptInvitation suma al usuario a la org con el rol de la invitación y
// la consume. Ya miembro → ErrConflict (la invitación queda disponible).
func (s *Service) AcceptInvitation(ctx context.Context, token, userID string) (*domain.Organization, error) {
	if userID == "" {
		return nil, domain.ErrUnauthorized
	}
	inv, err := s.invites.Accept(ctx, hashInvitationToken(token), userID, s.now())
	if err != nil {
		return nil, err
	}
	org, err := s.orgs.FindByID(ctx, inv.OrgID)
	if err != nil {
		return nil, err
	}
	org.Role = inv.Role
	return org, nil
}

func hashInvitationToken(token string) string {
	sum := sha256.Sum256([]byte(token))
	return hex.EncodeToString(sum[:])
}

// --- internos ---

// requireRole verifica la membership del actor contra el mínimo requerido y
// devuelve la org con el rol del actor. No miembro → ErrForbidden (aunque
// tampoco filtra si la org existe: no es información del cliente).
func (s *Service) requireRole(ctx context.Context, orgID, userID string, min domain.Role) (*domain.Organization, *domain.Membership, error) {
	if userID == "" {
		return nil, nil, domain.ErrUnauthorized
	}
	m, err := s.members.FindByOrgAndUser(ctx, orgID, userID)
	if err != nil {
		if errors.Is(err, domain.ErrNotFound) {
			return nil, nil, fmt.Errorf("%w: no eres miembro de la organización", domain.ErrForbidden)
		}
		return nil, nil, err
	}
	if !m.Role.AtLeast(min) {
		return nil, nil, fmt.Errorf("%w: se requiere rol %s (tienes %s)", domain.ErrForbidden, min, m.Role)
	}

	org, err := s.orgs.FindByID(ctx, orgID)
	if err != nil {
		return nil, nil, err
	}
	org.Role = m.Role
	return org, m, nil
}

// slugify deriva un slug URL-amigable del nombre (para la creación sin slug).
// stripAccents quita los diacríticos ("Ñandú" → "Nandu").
func stripAccents(s string) string {
	var b strings.Builder
	for _, r := range norm.NFD.String(s) {
		if !unicode.Is(unicode.Mn, r) {
			b.WriteRune(r)
		}
	}
	return b.String()
}

func slugify(name string) string {
	var b strings.Builder
	prevDash := false
	for _, r := range strings.ToLower(stripAccents(name)) {
		isAlnum := (r >= 'a' && r <= 'z') || (r >= '0' && r <= '9')
		if isAlnum {
			b.WriteRune(r)
			prevDash = false
			continue
		}
		if !prevDash && b.Len() > 0 {
			b.WriteRune('-')
			prevDash = true
		}
	}
	return strings.Trim(b.String(), "-")
}
