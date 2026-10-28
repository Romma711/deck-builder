package org

import (
	"context"
	"errors"
	"fmt"
	"testing"

	"github.com/Romma711/deck-builder/internal/domain"
	"github.com/Romma711/deck-builder/internal/ports"
	"github.com/google/uuid"
)

// --- Fakes de los puertos ---

// fakeMembershipRepo implementa ports.MembershipRepository.
type fakeMembershipRepo struct {
	byKey map[string]domain.Membership // "orgID|userID"
}

func newFakeMembershipRepo() *fakeMembershipRepo {
	return &fakeMembershipRepo{byKey: map[string]domain.Membership{}}
}

func key(orgID, userID string) string { return orgID + "|" + userID }

func (f *fakeMembershipRepo) Add(_ context.Context, m *domain.Membership) error {
	if _, err := uuid.Parse(m.OrgID); err != nil {
		return domain.ErrInvalidID
	}
	if _, err := uuid.Parse(m.UserID); err != nil {
		return domain.ErrInvalidID
	}
	if _, ok := f.byKey[key(m.OrgID, m.UserID)]; ok {
		return domain.ErrConflict
	}
	f.byKey[key(m.OrgID, m.UserID)] = *m
	return nil
}

func (f *fakeMembershipRepo) FindByOrgAndUser(_ context.Context, orgID, userID string) (*domain.Membership, error) {
	if _, err := uuid.Parse(orgID); err != nil {
		return nil, domain.ErrInvalidID
	}
	m, ok := f.byKey[key(orgID, userID)]
	if !ok {
		return nil, domain.ErrNotFound
	}
	return &m, nil
}

func (f *fakeMembershipRepo) UpdateRole(_ context.Context, orgID, userID string, role domain.Role) error {
	m, ok := f.byKey[key(orgID, userID)]
	if !ok {
		return domain.ErrNotFound
	}
	m.Role = role
	f.byKey[key(orgID, userID)] = m
	return nil
}

func (f *fakeMembershipRepo) Remove(_ context.Context, orgID, userID string) error {
	if _, ok := f.byKey[key(orgID, userID)]; !ok {
		return domain.ErrNotFound
	}
	delete(f.byKey, key(orgID, userID))
	return nil
}

func (f *fakeMembershipRepo) CountOwners(_ context.Context, orgID string) (int, error) {
	count := 0
	for _, m := range f.byKey {
		if m.OrgID == orgID && m.Role == domain.RoleOwner {
			count++
		}
	}
	return count, nil
}

func (f *fakeMembershipRepo) ListMembersByOrg(_ context.Context, orgID string) ([]ports.MemberInfo, error) {
	var out []ports.MemberInfo
	for _, m := range f.byKey {
		if m.OrgID == orgID {
			out = append(out, ports.MemberInfo{UserID: m.UserID, Role: m.Role, JoinedAt: m.CreatedAt})
		}
	}
	return out, nil
}

// fakeOrgRepo implementa ports.OrganizationRepository. Create replica el
// contrato del real: org + membership owner ATÓMICOS (aquí, en el mismo fake).
type fakeOrgRepo struct {
	orgs    map[string]*domain.Organization
	bySlug  map[string]string
	nextID  int
	members *fakeMembershipRepo
}

func newFakeOrgRepo(members *fakeMembershipRepo) *fakeOrgRepo {
	return &fakeOrgRepo{
		orgs: map[string]*domain.Organization{}, bySlug: map[string]string{},
		members: members,
	}
}

func (f *fakeOrgRepo) Create(_ context.Context, org *domain.Organization, ownerID string) error {
	if _, err := uuid.Parse(ownerID); err != nil {
		return domain.ErrInvalidID
	}
	if _, ok := f.bySlug[org.Slug]; ok {
		return fmt.Errorf("%w: slug ya en uso: %s", domain.ErrConflict, org.Slug)
	}
	f.nextID++
	org.ID = fmt.Sprintf("10000000-0000-7000-8000-%012d", f.nextID)
	f.orgs[org.ID] = org
	f.bySlug[org.Slug] = org.ID

	// Invariante: la org no sale sin owner.
	return f.members.Add(context.Background(), &domain.Membership{
		OrgID: org.ID, UserID: ownerID, Role: domain.RoleOwner, CreatedAt: org.CreatedAt,
	})
}

func (f *fakeOrgRepo) FindByID(_ context.Context, id string) (*domain.Organization, error) {
	org, ok := f.orgs[id]
	if !ok {
		return nil, domain.ErrNotFound
	}
	clone := *org
	return &clone, nil
}

func (f *fakeOrgRepo) FindBySlug(_ context.Context, slug string) (*domain.Organization, error) {
	id, ok := f.bySlug[slug]
	if !ok {
		return nil, domain.ErrNotFound
	}
	clone := *f.orgs[id]
	return &clone, nil
}

func (f *fakeOrgRepo) ListByUser(_ context.Context, userID string) ([]domain.Organization, error) {
	var out []domain.Organization
	for _, org := range f.orgs {
		if m, ok := f.members.byKey[key(org.ID, userID)]; ok {
			clone := *org
			clone.Role = m.Role
			out = append(out, clone)
		}
	}
	return out, nil
}

func (f *fakeOrgRepo) Update(_ context.Context, org *domain.Organization) error {
	if _, ok := f.orgs[org.ID]; !ok {
		return domain.ErrNotFound
	}
	if existing, ok := f.bySlug[org.Slug]; ok && existing != org.ID {
		return fmt.Errorf("%w: slug ya en uso: %s", domain.ErrConflict, org.Slug)
	}
	f.orgs[org.ID] = org
	return nil
}

func (f *fakeOrgRepo) Delete(_ context.Context, id string) error {
	if _, ok := f.orgs[id]; !ok {
		return domain.ErrNotFound
	}
	delete(f.orgs, id)
	// Cascada de memberships (como el FK real).
	for k, m := range f.members.byKey {
		if m.OrgID == id {
			delete(f.members.byKey, k)
		}
	}
	return nil
}

// fakeUserRepo implementa ports.UserRepository (solo FindByUsername es
// relevante aquí; el resto responde genéricamente).
type fakeUserRepo struct {
	byName map[string]*domain.User
}

func newFakeUserRepo(users ...*domain.User) *fakeUserRepo {
	f := &fakeUserRepo{byName: map[string]*domain.User{}}
	for _, u := range users {
		f.byName[u.Username] = u
	}
	return f
}

func (f *fakeUserRepo) Create(_ context.Context, _ *domain.User) error { return nil }

func (f *fakeUserRepo) FindByUsername(_ context.Context, name string) (*domain.User, error) {
	u, ok := f.byName[name]
	if !ok {
		return nil, domain.ErrNotFound
	}
	return u, nil
}

func (f *fakeUserRepo) FindByID(_ context.Context, id string) (*domain.User, error) {
	for _, u := range f.byName {
		if u.ID == id {
			return u, nil
		}
	}
	return nil, domain.ErrNotFound
}

func (f *fakeUserRepo) Update(_ context.Context, _ *domain.User) error { return nil }
func (f *fakeUserRepo) Delete(_ context.Context, id string) error {
	for name, u := range f.byName {
		if u.ID == id {
			delete(f.byName, name)
		}
	}
	return nil
}

// --- Harness ---

const (
	userOwner    = "aaaaaaaa-aaaa-aaaa-aaaa-aaaaaaaaaaaa"
	userAdmin    = "bbbbbbbb-bbbb-bbbb-bbbb-bbbbbbbbbbbb"
	userMember   = "cccccccc-cccc-cccc-cccc-cccccccccccc"
	userOutsider = "dddddddd-dddd-dddd-dddd-dddddddddddd"
)

// newHarness crea el servicio con una org "Tienda Beta" ya creada por userOwner,
// y mete a userAdmin (admin) y userMember (member) en ella. userOutsider
// EXISTE como usuario pero NO es miembro.
func newHarness(t *testing.T) (*Service, *fakeOrgRepo, *fakeMembershipRepo, string) {
	t.Helper()

	members := newFakeMembershipRepo()
	orgRepo := newFakeOrgRepo(members)
	users := newFakeUserRepo(
		&domain.User{ID: userOwner, Username: "owner", Email: "owner@x.dev"},
		&domain.User{ID: userAdmin, Username: "admin", Email: "admin@x.dev"},
		&domain.User{ID: userMember, Username: "member", Email: "member@x.dev"},
		&domain.User{ID: userOutsider, Username: "outsider", Email: "out@x.dev"},
	)
	svc := NewService(orgRepo, members, users)

	org, err := svc.Create(context.Background(), userOwner, CreateInput{Name: "Tienda Beta"})
	if err != nil {
		t.Fatalf("sembrar org: %v", err)
	}
	if err := members.Add(context.Background(), &domain.Membership{
		OrgID: org.ID, UserID: userAdmin, Role: domain.RoleAdmin,
	}); err != nil {
		t.Fatalf("sembrar admin: %v", err)
	}
	if err := members.Add(context.Background(), &domain.Membership{
		OrgID: org.ID, UserID: userMember, Role: domain.RoleMember,
	}); err != nil {
		t.Fatalf("sembrar member: %v", err)
	}
	return svc, orgRepo, members, org.ID
}

// --- Organización ---

func TestCreate_GeneraSlugYOwner(t *testing.T) {
	members := newFakeMembershipRepo()
	svc := NewService(newFakeOrgRepo(members), members, newFakeUserRepo())

	org, err := svc.Create(context.Background(), userOwner, CreateInput{Name: "Mi Tienda Gaming"})
	if err != nil {
		t.Fatalf("Create falló: %v", err)
	}
	if org.Slug != "mi-tienda-gaming" {
		t.Errorf("slug = %q, esperaba mi-tienda-gaming (derivado del nombre)", org.Slug)
	}
	if org.Role != domain.RoleOwner {
		t.Errorf("Role = %q, esperaba owner", org.Role)
	}
	m, err := members.FindByOrgAndUser(context.Background(), org.ID, userOwner)
	if err != nil {
		t.Fatalf("la org no quedó con su membership owner: %v", err)
	}
	if m.Role != domain.RoleOwner {
		t.Errorf("rol del creador = %q, esperaba owner", m.Role)
	}
}

func TestCreate_NombreCorto_Validation(t *testing.T) {
	members := newFakeMembershipRepo()
	svc := NewService(newFakeOrgRepo(members), members, newFakeUserRepo())

	_, err := svc.Create(context.Background(), userOwner, CreateInput{Name: "a"})
	if !errors.Is(err, domain.ErrValidation) {
		t.Fatalf("err = %v, esperaba ErrValidation", err)
	}
}

func TestCreate_SlugConMayusculas_Validation(t *testing.T) {
	members := newFakeMembershipRepo()
	svc := NewService(newFakeOrgRepo(members), members, newFakeUserRepo())

	_, err := svc.Create(context.Background(), userOwner, CreateInput{Name: "Tienda X", Slug: "Tienda X!"})
	if !errors.Is(err, domain.ErrValidation) {
		t.Fatalf("err = %v, esperaba ErrValidation", err)
	}
}

func TestCreate_SlugDuplicado_Conflict(t *testing.T) {
	members := newFakeMembershipRepo()
	svc := NewService(newFakeOrgRepo(members), members, newFakeUserRepo())

	if _, err := svc.Create(context.Background(), userOwner, CreateInput{Name: "Otra", Slug: "tienda"}); err != nil {
		t.Fatalf("primera Create falló: %v", err)
	}
	_, err := svc.Create(context.Background(), userOwner, CreateInput{Name: "Copia", Slug: "tienda"})
	if !errors.Is(err, domain.ErrConflict) {
		t.Fatalf("err = %v, esperaba ErrConflict", err)
	}
}

func TestGet_NoMiembro_Forbidden(t *testing.T) {
	svc, _, _, orgID := newHarness(t)

	_, err := svc.Get(context.Background(), orgID, userOutsider)
	if !errors.Is(err, domain.ErrForbidden) {
		t.Fatalf("err = %v, esperaba ErrForbidden", err)
	}
}

func TestGet_Miembro_DevuelveRol(t *testing.T) {
	svc, _, _, orgID := newHarness(t)

	org, err := svc.Get(context.Background(), orgID, userAdmin)
	if err != nil {
		t.Fatalf("Get falló: %v", err)
	}
	if org.Role != domain.RoleAdmin {
		t.Errorf("Role = %q, esperaba admin", org.Role)
	}
}

func TestGet_OrgInexistente_ForbiddenSinFiltrar(t *testing.T) {
	svc, _, _, _ := newHarness(t)

	// No miembro + org inexistente → también 403 (no se filtra existencia).
	_, err := svc.Get(context.Background(), "eeeeeeee-eeee-eeee-eeee-eeeeeeeeeeee", userOwner)
	if !errors.Is(err, domain.ErrForbidden) {
		t.Fatalf("err = %v, esperaba ErrForbidden", err)
	}
}

func TestGet_OrgIdBasura_InvalidID(t *testing.T) {
	svc, _, _, _ := newHarness(t)

	_, err := svc.Get(context.Background(), "basura", userOwner)
	if !errors.Is(err, domain.ErrInvalidID) {
		t.Fatalf("err = %v, esperaba ErrInvalidID", err)
	}
}

func TestListMine_ConRoles(t *testing.T) {
	svc, _, _, orgID := newHarness(t)

	orgs, err := svc.ListMine(context.Background(), userMember)
	if err != nil {
		t.Fatalf("ListMine falló: %v", err)
	}
	if len(orgs) != 1 || orgs[0].ID != orgID {
		t.Fatalf("orgs = %+v, esperaba exactamente la org sembrada", orgs)
	}
	if orgs[0].Role != domain.RoleMember {
		t.Errorf("Role = %q, esperaba member", orgs[0].Role)
	}
}

func TestUpdate_SoloOwner(t *testing.T) {
	svc, _, _, orgID := newHarness(t)

	// Un admin NO puede renombrar.
	_, err := svc.Update(context.Background(), orgID, userAdmin, UpdateInput{Name: "Nuevo", Slug: "nuevo"})
	if !errors.Is(err, domain.ErrForbidden) {
		t.Fatalf("err = %v (admin), esperaba ErrForbidden", err)
	}

	// El owner sí.
	org, err := svc.Update(context.Background(), orgID, userOwner, UpdateInput{Name: "Nuevo", Slug: "nuevo"})
	if err != nil {
		t.Fatalf("Update del owner falló: %v", err)
	}
	if org.Name != "Nuevo" || org.Slug != "nuevo" {
		t.Errorf("org = %+v, esperaba Nombre Nuevo / slug nuevo", org)
	}
}

func TestDelete_SoloOwner(t *testing.T) {
	svc, _, _, orgID := newHarness(t)

	if err := svc.Delete(context.Background(), orgID, userAdmin); !errors.Is(err, domain.ErrForbidden) {
		t.Fatalf("err = %v (admin), esperaba ErrForbidden", err)
	}
	if err := svc.Delete(context.Background(), orgID, userOwner); err != nil {
		t.Fatalf("Delete del owner falló: %v", err)
	}
	// Ya borrada: la membership cae en cascada → el owner ya no es miembro.
	if err := svc.Delete(context.Background(), orgID, userOwner); !errors.Is(err, domain.ErrForbidden) {
		t.Fatalf("err = %v (org ya borrada), esperaba ErrForbidden (sin membresía)", err)
	}
}

// --- Miembros: altas ---

func TestAddMember_Ok(t *testing.T) {
	svc, _, members, orgID := newHarness(t)

	member, err := svc.AddMember(context.Background(), orgID, userOwner, "outsider", domain.RoleMember)
	if err != nil {
		t.Fatalf("AddMember falló: %v", err)
	}
	if member.UserID != userOutsider || member.Role != domain.RoleMember {
		t.Errorf("member = %+v, esperaba user outsider / role member", member)
	}
	if _, err := members.FindByOrgAndUser(context.Background(), orgID, userOutsider); err != nil {
		t.Errorf("membership no persistida: %v", err)
	}
}

func TestAddMember_UsuarioDesconocido_Validation(t *testing.T) {
	svc, _, _, orgID := newHarness(t)

	_, err := svc.AddMember(context.Background(), orgID, userOwner, "no-existe", "")
	if !errors.Is(err, domain.ErrValidation) {
		t.Fatalf("err = %v, esperaba ErrValidation con detalle de usuario", err)
	}
}

func TestAddMember_YaEsMiembro_Conflict(t *testing.T) {
	svc, _, _, orgID := newHarness(t)

	_, err := svc.AddMember(context.Background(), orgID, userOwner, "admin", domain.RoleMember)
	if !errors.Is(err, domain.ErrConflict) {
		t.Fatalf("err = %v, esperaba ErrConflict", err)
	}
}

func TestAddMember_NoMiembroActor_Forbidden(t *testing.T) {
	svc, _, _, orgID := newHarness(t)

	_, err := svc.AddMember(context.Background(), orgID, userOutsider, "member", domain.RoleMember)
	if !errors.Is(err, domain.ErrForbidden) {
		t.Fatalf("err = %v, esperaba ErrForbidden (outsider no es miembro)", err)
	}
}

func TestAddMember_AdminSoloOtorgaMember(t *testing.T) {
	svc, _, _, orgID := newHarness(t)

	// El admin NO puede otorgar rol admin.
	_, err := svc.AddMember(context.Background(), orgID, userAdmin, "outsider", domain.RoleAdmin)
	if !errors.Is(err, domain.ErrForbidden) {
		t.Fatalf("err = %v (admin otorgando admin), esperaba ErrForbidden", err)
	}

	// Sí puede otorgar member.
	if _, err := svc.AddMember(context.Background(), orgID, userAdmin, "outsider", domain.RoleMember); err != nil {
		t.Fatalf("admin otorgando member falló: %v", err)
	}
}

// --- Miembros: roles ---

func TestChangeRole_SoloOwner(t *testing.T) {
	svc, _, _, orgID := newHarness(t)

	// El admin no cambia roles (ni siquiera a member).
	err := svc.ChangeRole(context.Background(), orgID, userAdmin, userMember, domain.RoleAdmin)
	if !errors.Is(err, domain.ErrForbidden) {
		t.Fatalf("err = %v (admin cambiando rol), esperaba ErrForbidden", err)
	}

	// El owner promueve member → admin.
	if err := svc.ChangeRole(context.Background(), orgID, userOwner, userMember, domain.RoleAdmin); err != nil {
		t.Fatalf("ChangeRole del owner falló: %v", err)
	}
}

func TestChangeRole_UltimoOwner_NoSeDegrada(t *testing.T) {
	svc, _, _, orgID := newHarness(t)

	err := svc.ChangeRole(context.Background(), orgID, userOwner, userOwner, domain.RoleMember)
	if !errors.Is(err, domain.ErrConflict) {
		t.Fatalf("err = %v, esperaba ErrConflict (último owner)", err)
	}
}

func TestChangeRole_ConSegundoOwner_SiSeDegrada(t *testing.T) {
	svc, _, members, orgID := newHarness(t)

	// Segundo owner → el primero ya puede degradarse.
	if err := members.UpdateRole(context.Background(), orgID, userAdmin, domain.RoleOwner); err != nil {
		t.Fatalf("sembrar segundo owner: %v", err)
	}
	if err := svc.ChangeRole(context.Background(), orgID, userAdmin, userOwner, domain.RoleAdmin); err != nil {
		t.Fatalf("degradación con 2 owners falló: %v", err)
	}
}

func TestChangeRole_NoEsMiembro_NotFound(t *testing.T) {
	svc, _, _, orgID := newHarness(t)

	err := svc.ChangeRole(context.Background(), orgID, userOwner, userOutsider, domain.RoleAdmin)
	if !errors.Is(err, domain.ErrNotFound) {
		t.Fatalf("err = %v, esperaba ErrNotFound (target sin membership)", err)
	}
}

func TestChangeRole_RolBasura_Validation(t *testing.T) {
	svc, _, _, orgID := newHarness(t)

	err := svc.ChangeRole(context.Background(), orgID, userOwner, userMember, domain.Role("superuser"))
	if !errors.Is(err, domain.ErrValidation) {
		t.Fatalf("err = %v, esperaba ErrValidation", err)
	}
}

// --- Miembros: bajas ---

func TestRemoveMember_AutoSalida_Ok(t *testing.T) {
	svc, _, members, orgID := newHarness(t)

	if err := svc.RemoveMember(context.Background(), orgID, userMember, userMember); err != nil {
		t.Fatalf("auto-salida de member falló: %v", err)
	}
	if _, err := members.FindByOrgAndUser(context.Background(), orgID, userMember); !errors.Is(err, domain.ErrNotFound) {
		t.Errorf("membership persiste tras salir: %v", err)
	}
}

func TestRemoveMember_UltimoOwner_NoSale(t *testing.T) {
	svc, _, _, orgID := newHarness(t)

	err := svc.RemoveMember(context.Background(), orgID, userOwner, userOwner)
	if !errors.Is(err, domain.ErrConflict) {
		t.Fatalf("err = %v, esperaba ErrConflict (último owner no puede salirse)", err)
	}
}

func TestRemoveMember_AdminQuitaMember_Ok(t *testing.T) {
	svc, _, _, orgID := newHarness(t)

	if err := svc.RemoveMember(context.Background(), orgID, userAdmin, userMember); err != nil {
		t.Fatalf("admin quitando member falló: %v", err)
	}
}

func TestRemoveMember_AdminNoQuitaAdminNiOwner_Forbidden(t *testing.T) {
	svc, _, members, orgID := newHarness(t)

	// userMember pasa a admin (sembrado directo, como haría el owner).
	if err := members.UpdateRole(context.Background(), orgID, userMember, domain.RoleAdmin); err != nil {
		t.Fatalf("sembrar segundo admin: %v", err)
	}
	err := svc.RemoveMember(context.Background(), orgID, userAdmin, userMember)
	if !errors.Is(err, domain.ErrForbidden) {
		t.Fatalf("err = %v (admin quitando admin), esperaba ErrForbidden", err)
	}

	// Admin quitando al owner (único): la guardia de último owner corre antes.
	err = svc.RemoveMember(context.Background(), orgID, userAdmin, userOwner)
	if !errors.Is(err, domain.ErrConflict) {
		t.Fatalf("err = %v, esperaba ErrConflict (último owner)", err)
	}
}

func TestRemoveMember_OwnerQuitaAdmin_Ok(t *testing.T) {
	svc, _, _, orgID := newHarness(t)

	if err := svc.RemoveMember(context.Background(), orgID, userOwner, userAdmin); err != nil {
		t.Fatalf("owner quitando admin falló: %v", err)
	}
}

func TestRemoveMember_NoMiembroActor_Forbidden(t *testing.T) {
	svc, _, _, orgID := newHarness(t)

	// Outsider intentando auto-salirse de una org a la que no pertenece:
	// el target no es miembro → NotFound (y jamás 204).
	err := svc.RemoveMember(context.Background(), orgID, userOutsider, userOutsider)
	if !errors.Is(err, domain.ErrNotFound) {
		t.Fatalf("err = %v, esperaba ErrNotFound", err)
	}
}

// --- Lectura de miembros ---

func TestListMembers_RequiereMembresia(t *testing.T) {
	svc, _, _, orgID := newHarness(t)

	if _, err := svc.ListMembers(context.Background(), orgID, userOutsider); !errors.Is(err, domain.ErrForbidden) {
		t.Fatalf("err = %v, esperaba ErrForbidden", err)
	}
	members, err := svc.ListMembers(context.Background(), orgID, userMember)
	if err != nil {
		t.Fatalf("ListMembers (member) falló: %v", err)
	}
	if len(members) != 3 {
		t.Errorf("cantidad = %d, esperaba 3 (owner, admin, member)", len(members))
	}
}

// --- Jerarquía de roles (unidad pura) ---

func TestRoleAtLeast_Jerarquia(t *testing.T) {
	cases := []struct {
		have domain.Role
		min  domain.Role
		want bool
	}{
		{domain.RoleOwner, domain.RoleMember, true},
		{domain.RoleOwner, domain.RoleOwner, true},
		{domain.RoleAdmin, domain.RoleMember, true},
		{domain.RoleAdmin, domain.RoleOwner, false},
		{domain.RoleMember, domain.RoleAdmin, false},
		{domain.RoleMember, domain.RoleMember, true},
		{domain.Role("x"), domain.RoleMember, false}, // desconocido: falla cerrada
		{domain.RoleOwner, domain.Role("x"), false},
	}
	for _, tc := range cases {
		if got := tc.have.AtLeast(tc.min); got != tc.want {
			t.Errorf("%q.AtLeast(%q) = %v, esperaba %v", tc.have, tc.min, got, tc.want)
		}
	}
}
