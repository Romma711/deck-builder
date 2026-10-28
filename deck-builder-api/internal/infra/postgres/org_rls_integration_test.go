package postgres

// Tests de INTEGRACIÓN de la Fase 5 (SaaS):
//  1. CRUD de organizaciones y memberships.
//  2. Row-Level Security: aislamiento de mazos privados y de org con un rol
//     NO superusuario (el usuario "test" del embebido es superusuario y RLS
//     lo ignora, así que estos tests abren una segunda conexión como
//     "deck_rls_login" para probar las políticas de verdad).

import (
	"context"
	"errors"
	"net/url"
	"os"
	"testing"
	"time"

	"github.com/Romma711/deck-builder/internal/domain"
	"github.com/Romma711/deck-builder/internal/ports"
	"gorm.io/gorm"
)

// --- helpers ---

func integrationUser(t *testing.T, db *gorm.DB, username string) *domain.User {
	t.Helper()
	now := time.Now().UTC()
	u := &domain.User{
		Username: username, Email: username + "@example.com", PasswordHash: "hash",
		CreatedAt: now, UpdatedAt: now,
	}
	if err := NewUserRepository(db).Create(context.Background(), u); err != nil {
		t.Fatalf("crear usuario %s: %v", username, err)
	}
	return u
}

func integrationOrg(t *testing.T, db *gorm.DB, owner *domain.User, slug, name string) *domain.Organization {
	t.Helper()
	now := time.Now().UTC()
	org := &domain.Organization{Slug: slug, Name: name, CreatedAt: now, UpdatedAt: now}
	if err := NewOrganizationRepository(db).Create(context.Background(), org, owner.ID); err != nil {
		t.Fatalf("crear org %s: %v", slug, err)
	}
	return org
}

// seedDeck crea un mazo directo vía repositorio (como superusuario: RLS no aplica).
func seedDeck(t *testing.T, db *gorm.DB, ownerID, name string, isPrivate bool, orgID string) *domain.Deck {
	t.Helper()
	now := time.Now().UTC()
	deck := &domain.Deck{
		OwnerID: ownerID, GameID: gameIDByCode(t, db, "mtg"),
		OrgID: orgID, IsPrivate: isPrivate,
		Name: name, DeckSize: 60, Format: "standard",
		CreatedAt: now, UpdatedAt: now,
	}
	if err := NewDeckRepository(db).Create(context.Background(), deck); err != nil {
		t.Fatalf("sembrar mazo %s: %v", name, err)
	}
	return deck
}

// --- 1. CRUD de organizaciones ---

func TestIntegration_OrganizationRepository_CRUD(t *testing.T) {
	db := setupTestDB(t)
	orgRepo := NewOrganizationRepository(db)
	memRepo := NewMembershipRepository(db)
	ctx := context.Background()

	owner := integrationUser(t, db, "owner_org")
	now := time.Now().UTC()

	// Create: org + membership owner atómicos.
	org := &domain.Organization{Slug: "tienda-beta", Name: "Tienda Beta", CreatedAt: now, UpdatedAt: now}
	if err := orgRepo.Create(ctx, org, owner.ID); err != nil {
		t.Fatalf("Create: %v", err)
	}
	if org.ID == "" {
		t.Fatal("Create no asignó el ID")
	}
	m, err := memRepo.FindByOrgAndUser(ctx, org.ID, owner.ID)
	if err != nil {
		t.Fatalf("la org no quedó con su membership owner: %v", err)
	}
	if m.Role != domain.RoleOwner {
		t.Errorf("rol = %q, esperaba owner", m.Role)
	}

	// Slug duplicado → ErrConflict (y NO deja org huérfana).
	dup := &domain.Organization{Slug: "tienda-beta", Name: "Copia", CreatedAt: now, UpdatedAt: now}
	if err := orgRepo.Create(ctx, dup, owner.ID); !errors.Is(err, domain.ErrConflict) {
		t.Fatalf("slug duplicado: err = %v, esperaba ErrConflict", err)
	}

	// FindByID / FindBySlug.
	byID, err := orgRepo.FindByID(ctx, org.ID)
	if err != nil || byID.Name != "Tienda Beta" {
		t.Fatalf("FindByID = %+v, %v", byID, err)
	}
	if _, err := orgRepo.FindBySlug(ctx, "tienda-beta"); err != nil {
		t.Fatalf("FindBySlug: %v", err)
	}
	if _, err := orgRepo.FindByID(ctx, "eeeeeeee-eeee-eeee-eeee-eeeeeeeeeeee"); !errors.Is(err, domain.ErrNotFound) {
		t.Fatalf("FindByID inexistente: err = %v, esperaba ErrNotFound", err)
	}
	if _, err := orgRepo.FindByID(ctx, "basura"); !errors.Is(err, domain.ErrInvalidID) {
		t.Fatalf("FindByID basura: err = %v, esperaba ErrInvalidID", err)
	}

	// ListByUser con el rol del consultante.
	orgs, err := orgRepo.ListByUser(ctx, owner.ID)
	if err != nil {
		t.Fatalf("ListByUser: %v", err)
	}
	if len(orgs) != 1 || orgs[0].ID != org.ID || orgs[0].Role != domain.RoleOwner {
		t.Fatalf("ListByUser = %+v, esperaba la org con rol owner", orgs)
	}

	// Update.
	org.Name = "Tienda Gamma"
	org.UpdatedAt = time.Now().UTC()
	if err := orgRepo.Update(ctx, org); err != nil {
		t.Fatalf("Update: %v", err)
	}
	renamed, _ := orgRepo.FindByID(ctx, org.ID)
	if renamed.Name != "Tienda Gamma" {
		t.Errorf("name tras update = %q", renamed.Name)
	}
	// Update de org inexistente → ErrNotFound.
	ghost := &domain.Organization{ID: "eeeeeeee-eeee-eeee-eeee-eeeeeeeeeeee", Slug: "ghost", Name: "Ghost"}
	if err := orgRepo.Update(ctx, ghost); !errors.Is(err, domain.ErrNotFound) {
		t.Fatalf("Update fantasma: err = %v, esperaba ErrNotFound", err)
	}

	// Delete: memberships caen en cascada.
	if err := orgRepo.Delete(ctx, org.ID); err != nil {
		t.Fatalf("Delete: %v", err)
	}
	if _, err := orgRepo.FindByID(ctx, org.ID); !errors.Is(err, domain.ErrNotFound) {
		t.Errorf("la org persiste tras Delete: %v", err)
	}
	if _, err := memRepo.FindByOrgAndUser(ctx, org.ID, owner.ID); !errors.Is(err, domain.ErrNotFound) {
		t.Errorf("la membership no cayó en cascada: %v", err)
	}
	if err := orgRepo.Delete(ctx, org.ID); !errors.Is(err, domain.ErrNotFound) {
		t.Errorf("Delete repetido: err = %v, esperaba ErrNotFound", err)
	}
}

// --- 2. CRUD de memberships ---

func TestIntegration_MembershipRepository_CRUD(t *testing.T) {
	db := setupTestDB(t)
	memRepo := NewMembershipRepository(db)
	ctx := context.Background()

	owner := integrationUser(t, db, "owner_mb")
	member := integrationUser(t, db, "member_mb")
	org := integrationOrg(t, db, owner, "mb-org", "MB Org")

	// Add.
	m := &domain.Membership{OrgID: org.ID, UserID: member.ID, Role: domain.RoleMember, CreatedAt: time.Now().UTC()}
	if err := memRepo.Add(ctx, m); err != nil {
		t.Fatalf("Add: %v", err)
	}

	// Duplicado → ErrConflict.
	if err := memRepo.Add(ctx, m); !errors.Is(err, domain.ErrConflict) {
		t.Fatalf("Add duplicado: err = %v, esperaba ErrConflict", err)
	}

	// Org inexistente (uuid válido) → ErrNotFound vía FK.
	orphan := &domain.Membership{
		OrgID: "eeeeeeee-eeee-eeee-eeee-eeeeeeeeeeee", UserID: member.ID, Role: domain.RoleMember,
	}
	if err := memRepo.Add(ctx, orphan); !errors.Is(err, domain.ErrNotFound) {
		t.Fatalf("Add a org inexistente: err = %v, esperaba ErrNotFound", err)
	}

	// UUID basura → ErrInvalidID.
	bad := &domain.Membership{OrgID: "basura", UserID: member.ID, Role: domain.RoleMember}
	if err := memRepo.Add(ctx, bad); !errors.Is(err, domain.ErrInvalidID) {
		t.Fatalf("Add basura: err = %v, esperaba ErrInvalidID", err)
	}

	// Find.
	found, err := memRepo.FindByOrgAndUser(ctx, org.ID, member.ID)
	if err != nil || found.Role != domain.RoleMember {
		t.Fatalf("FindByOrgAndUser = %+v, %v", found, err)
	}
	// La membership owner la creó orgRepo.Create (transacción atómica).
	if _, err := memRepo.FindByOrgAndUser(ctx, org.ID, owner.ID); err != nil {
		t.Fatalf("la membership owner no existe: %v", err)
	}

	// CountOwners: solo el creador.
	count, err := memRepo.CountOwners(ctx, org.ID)
	if err != nil || count != 1 {
		t.Fatalf("CountOwners = %d, %v; esperaba 1", count, err)
	}

	// UpdateRole.
	if err := memRepo.UpdateRole(ctx, org.ID, member.ID, domain.RoleAdmin); err != nil {
		t.Fatalf("UpdateRole: %v", err)
	}
	upd, _ := memRepo.FindByOrgAndUser(ctx, org.ID, member.ID)
	if upd.Role != domain.RoleAdmin {
		t.Errorf("rol tras update = %q, esperaba admin", upd.Role)
	}
	if err := memRepo.UpdateRole(ctx, org.ID, "eeeeeeee-eeee-eeee-eeee-eeeeeeeeeeee", domain.RoleMember); !errors.Is(err, domain.ErrNotFound) {
		t.Errorf("UpdateRole sin membership: err = %v, esperaba ErrNotFound", err)
	}

	// ListMembersByOrg con JOIN a users (username/email) sin N+1.
	list, err := memRepo.ListMembersByOrg(ctx, org.ID)
	if err != nil {
		t.Fatalf("ListMembersByOrg: %v", err)
	}
	if len(list) != 2 {
		t.Fatalf("cantidad = %d, esperaba 2 (owner + admin)", len(list))
	}
	byUser := map[string]string{}
	for _, mi := range list {
		byUser[mi.UserID] = mi.Username
	}
	if byUser[member.ID] != "member_mb" || byUser[owner.ID] != "owner_mb" {
		t.Errorf("usernames = %+v, esperaba member_mb/owner_mb", byUser)
	}

	// Remove.
	if err := memRepo.Remove(ctx, org.ID, member.ID); err != nil {
		t.Fatalf("Remove: %v", err)
	}
	if err := memRepo.Remove(ctx, org.ID, member.ID); !errors.Is(err, domain.ErrNotFound) {
		t.Errorf("Remove repetido: err = %v, esperaba ErrNotFound", err)
	}
	if _, err := memRepo.FindByOrgAndUser(ctx, org.ID, member.ID); !errors.Is(err, domain.ErrNotFound) {
		t.Errorf("la membership persiste tras Remove: %v", err)
	}
}

// --- 3. Row-Level Security con rol NO superusuario ---

const (
	rlsRole = "deck_rls_login"
	rlsPass = "rls_pass"
)

// ensureRestrictedRole crea un rol de BD sin superusuario con privilegios de
// lectura/escritura sobre las tablas (para que RLS realmente aplique).
func ensureRestrictedRole(t *testing.T, db *gorm.DB) {
	t.Helper()
	if err := db.Exec(`DO $$
BEGIN
	CREATE ROLE ` + rlsRole + ` LOGIN PASSWORD '` + rlsPass + `' NOSUPERUSER;
EXCEPTION WHEN duplicate_object THEN
	NULL;
END $$`).Error; err != nil {
		t.Skipf("sin privilegios para crear rol (¿conexión no superusuario?): %v", err)
	}
	if err := db.Exec("GRANT USAGE ON SCHEMA public TO " + rlsRole).Error; err != nil {
		t.Skipf("sin privilegios para GRANT: %v", err)
	}
	if err := db.Exec("GRANT SELECT, INSERT, UPDATE, DELETE ON ALL TABLES IN SCHEMA public TO " + rlsRole).Error; err != nil {
		t.Fatalf("GRANT tablas: %v", err)
	}
	t.Cleanup(func() {
		_ = db.Exec("DROP OWNED BY " + rlsRole).Error
		_ = db.Exec("DROP ROLE IF EXISTS " + rlsRole).Error
	})
}

// openRestrictedDB abre una segunda conexión como el rol restringido.
func openRestrictedDB(t *testing.T) *gorm.DB {
	t.Helper()
	raw := os.Getenv("TEST_DATABASE_URL")
	u, err := url.Parse(raw)
	if err != nil {
		t.Skipf("TEST_DATABASE_URL no es una URL parseable: %v", err)
	}
	u.User = url.UserPassword(rlsRole, rlsPass)

	db, err := Open(context.Background(), Options{DSN: u.String()})
	if err != nil {
		t.Fatalf("conectar como rol restringido: %v", err)
	}
	sqlDB, _ := db.DB()
	t.Cleanup(func() { _ = sqlDB.Close() })
	return db
}

func ctxAs(userID string) context.Context {
	if userID == "" {
		return context.Background()
	}
	return domain.WithUserID(context.Background(), userID)
}

func TestIntegration_RLS_VisibilidadDeMazos(t *testing.T) {
	db := setupTestDB(t)
	ensureRestrictedRole(t, db)

	// --- Escenario (sembrado como superusuario) ---
	// alice (dueña), bob (miembro de la org), carol (ajena).
	alice := integrationUser(t, db, "alice")
	bob := integrationUser(t, db, "bob")
	carol := integrationUser(t, db, "carol")

	org := integrationOrg(t, db, alice, "rls-tienda", "RLS Tienda")
	memRepo := NewMembershipRepository(db)
	if err := memRepo.Add(context.Background(), &domain.Membership{
		OrgID: org.ID, UserID: bob.ID, Role: domain.RoleMember, CreatedAt: time.Now().UTC(),
	}); err != nil {
		t.Fatalf("sembrar member: %v", err)
	}

	alicePrivate := seedDeck(t, db, alice.ID, "Privado de Alice", true, "")
	alicePublic := seedDeck(t, db, alice.ID, "Publico de Alice", false, "")
	orgPrivate := seedDeck(t, db, alice.ID, "Privado de la Org", true, org.ID)

	// --- Conexión real (no superusuario) ---
	rlsDB := openRestrictedDB(t)
	deckRepo := NewDeckRepository(rlsDB)

	// 1. La dueña ve su mazo privado.
	deck, err := deckRepo.FindByID(ctxAs(alice.ID), alicePrivate.ID)
	if err != nil || !deck.IsPrivate {
		t.Fatalf("dueña no ve su privado: deck=%+v err=%v", deck, err)
	}

	// 2. Otro usuario NO ve el privado ajeno (404, sin filtrar existencia).
	if _, err := deckRepo.FindByID(ctxAs(bob.ID), alicePrivate.ID); !errors.Is(err, domain.ErrNotFound) {
		t.Fatalf("bob ve el privado de alice: err = %v, esperaba ErrNotFound", err)
	}

	// 3. El público sí lo ven todos.
	if _, err := deckRepo.FindByID(ctxAs(bob.ID), alicePublic.ID); err != nil {
		t.Fatalf("bob no ve el público: %v", err)
	}
	if _, err := deckRepo.FindByID(context.Background(), alicePublic.ID); err != nil {
		t.Fatalf("anónimo no ve el público: %v", err)
	}

	// 4. Anónimo: jamás ve privados (ni de alice ni de la org).
	if _, err := deckRepo.FindByID(context.Background(), alicePrivate.ID); !errors.Is(err, domain.ErrNotFound) {
		t.Fatalf("anónimo ve el privado de alice: err = %v", err)
	}
	if _, err := deckRepo.FindByID(context.Background(), orgPrivate.ID); !errors.Is(err, domain.ErrNotFound) {
		t.Fatalf("anónimo ve el privado de la org: err = %v", err)
	}

	// 5. El privado de la org: lo ve un MIEMBRO, no un ajeno.
	if _, err := deckRepo.FindByID(ctxAs(bob.ID), orgPrivate.ID); err != nil {
		t.Fatalf("bob (miembro) no ve el privado de la org: %v", err)
	}
	if _, err := deckRepo.FindByID(ctxAs(carol.ID), orgPrivate.ID); !errors.Is(err, domain.ErrNotFound) {
		t.Fatalf("carol (ajena) ve el privado de la org: err = %v, esperaba ErrNotFound", err)
	}

	// 6. Listados acotados por RLS:
	//    anónimo → solo el público (1); bob → público + org privado (2).
	anon, err := deckRepo.Find(context.Background(), ports.DeckFilter{})
	if err != nil {
		t.Fatalf("Find anónimo: %v", err)
	}
	if len(anon) != 1 || anon[0].ID != alicePublic.ID {
		t.Fatalf("Find anónimo = %d mazos, esperaba solo el público", len(anon))
	}
	seenByBob, err := deckRepo.Find(ctxAs(bob.ID), ports.DeckFilter{})
	if err != nil {
		t.Fatalf("Find bob: %v", err)
	}
	if len(seenByBob) != 2 {
		ids := []string{}
		for _, d := range seenByBob {
			ids = append(ids, d.Name)
		}
		t.Fatalf("Find bob = %v, esperaba público + org privado (2)", ids)
	}
}

func TestIntegration_RLS_EscrituraBloqueada(t *testing.T) {
	db := setupTestDB(t)
	ensureRestrictedRole(t, db)

	alice := integrationUser(t, db, "alice_w")
	bob := integrationUser(t, db, "bob_w")

	aliceDeck := seedDeck(t, db, alice.ID, "Privado Alice", true, "")
	alicePublic := seedDeck(t, db, alice.ID, "Publico Alice", false, "")

	rlsDB := openRestrictedDB(t)
	deckRepo := NewDeckRepository(rlsDB)
	ctx := context.Background()

	// 1. bob NO puede actualizar el mazo de alice (0 filas → 404).
	toUpdate := *aliceDeck
	toUpdate.Name = "Hackeado"
	if err := deckRepo.Update(ctxAs(bob.ID), &toUpdate); !errors.Is(err, domain.ErrNotFound) {
		t.Fatalf("Update ajeno: err = %v, esperaba ErrNotFound (RLS filtra la fila)", err)
	}

	// 2. bob NO puede borrar el de alice.
	if err := deckRepo.Delete(ctxAs(bob.ID), aliceDeck.ID); !errors.Is(err, domain.ErrNotFound) {
		t.Fatalf("Delete ajeno: err = %v, esperaba ErrNotFound", err)
	}

	// 3. Ni siquiera el PÚBLICO: leer no es escribir (USING = solo dueño).
	toUpdate = *alicePublic
	toUpdate.Name = "Vandalizado"
	if err := deckRepo.Update(ctxAs(bob.ID), &toUpdate); !errors.Is(err, domain.ErrNotFound) {
		t.Fatalf("Update de público ajeno: err = %v, esperaba ErrNotFound", err)
	}

	// 4. bob NO puede crear un mazo a nombre de alice (WITH CHECK owner).
	steal := *aliceDeck
	steal.ID = ""
	steal.OwnerID = alice.ID
	steal.Name = "Mazo Robado"
	if err := deckRepo.Create(ctxAs(bob.ID), &steal); err == nil {
		t.Fatal("Create suplantando al dueño: esperaba violación de RLS (WITH CHECK)")
	}

	// 5. Tampoco anónimo (sin app.user_id → owner NULL ≠ cualquier uuid).
	anon := *aliceDeck
	anon.ID = ""
	anon.OwnerID = alice.ID
	anon.Name = "Mazo Anónimo"
	if err := deckRepo.Create(ctx, &anon); err == nil {
		t.Fatal("Create anónimo: esperaba violación de RLS (WITH CHECK)")
	}

	// 6. bob sí puede crear y gestionar SUS mazos (no rompimos lo normal).
	now := time.Now().UTC()
	bobDeck := &domain.Deck{
		OwnerID: bob.ID, GameID: gameIDByCode(t, db, "mtg"),
		Name: "Mazo de Bob", DeckSize: 60, Format: "standard", IsPrivate: true,
		CreatedAt: now, UpdatedAt: now,
	}
	if err := deckRepo.Create(ctxAs(bob.ID), bobDeck); err != nil {
		t.Fatalf("Create propio como bob: %v", err)
	}
	bobDeck.Name = "Mazo de Bob v2"
	bobDeck.UpdatedAt = time.Now().UTC()
	if err := deckRepo.Update(ctxAs(bob.ID), bobDeck); err != nil {
		t.Fatalf("Update propio como bob: %v", err)
	}
	if _, err := deckRepo.FindByID(ctxAs(alice.ID), bobDeck.ID); !errors.Is(err, domain.ErrNotFound) {
		t.Fatalf("alice ve el privado de bob: err = %v, esperaba ErrNotFound", err)
	}
	if err := deckRepo.Delete(ctxAs(bob.ID), bobDeck.ID); err != nil {
		t.Fatalf("Delete propio como bob: %v", err)
	}

	// 7. Las cartas del mazo NO se filtran a mano alzada (heredan visibilidad).
	//    Sembramos una carta real en el mazo privado de alice: sin RLS, la
	//    consulta directa a deck_cards vería 1 fila.
	gameID := gameIDByCode(t, db, "mtg")
	card, err := NewCardRepository(db).Upsert(ctx, &domain.Card{
		GameID: gameID, Name: "Sol Ring", SerialSet: "LEA",
		Image: "https://example.com/solring.png", Attributes: map[string]any{},
	})
	if err != nil {
		t.Fatalf("sembrar carta: %v", err)
	}
	if err := db.Exec(
		"INSERT INTO deck_cards (deck_id, card_id, quantity) VALUES (?, ?, 4)",
		aliceDeck.ID, card.ID,
	).Error; err != nil {
		t.Fatalf("sembrar deck_cards: %v", err)
	}

	var visibleForOwner int64
	if err := db.Raw(
		"SELECT COUNT(*) FROM deck_cards WHERE deck_id = ?", aliceDeck.ID,
	).Scan(&visibleForOwner).Error; err != nil {
		t.Fatalf("count como superusuario: %v", err)
	}
	if visibleForOwner != 1 {
		t.Fatalf("sanity: el superusuario ve %d filas, esperaba 1", visibleForOwner)
	}

	var leaked int64
	if err := rlsDB.Raw(
		"SELECT COUNT(*) FROM deck_cards WHERE deck_id = ?", aliceDeck.ID,
	).Scan(&leaked).Error; err != nil {
		t.Fatalf("consulta directa a deck_cards: %v", err)
	}
	if leaked != 0 {
		t.Errorf("deck_cards expuso %d filas bajo identidad ajena", leaked)
	}
}
