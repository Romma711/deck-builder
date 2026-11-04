package postgres

// Tests de INTEGRACIÓN contra PostgreSQL real.
//
// Se omiten automáticamente si no hay TEST_DATABASE_URL configurada:
//
//	TEST_DATABASE_URL=postgres://user:pass@localhost:5432/deckbuilder_test go test ./internal/infra/postgres/...
//
// La base debe estar vacía: el test aplica las migraciones desde cero.

import (
	"context"
	"errors"
	"os"
	"testing"
	"time"

	"github.com/Romma711/deck-builder/internal/domain"
	"github.com/Romma711/deck-builder/internal/ports"
	"gorm.io/gorm"
)

func setupTestDB(t *testing.T) *gorm.DB {
	t.Helper()

	dsn := os.Getenv("TEST_DATABASE_URL")
	if dsn == "" {
		t.Skip("TEST_DATABASE_URL no configurada: omitiendo test de integración")
	}

	db, err := Open(context.Background(), Options{DSN: dsn})
	if err != nil {
		t.Fatalf("conectar a la base de test: %v", err)
	}

	if err := Migrate(db); err != nil {
		t.Fatalf("migrar la base de test: %v", err)
	}

	// Limpiamos datos entre tests (orden inverso a las dependencias FK).
	// games/formats son seeds de la migración: se conservan.
	sqlDB, _ := db.DB()
	t.Cleanup(func() { _ = sqlDB.Close() })
	for _, table := range []string{"deck_cards", "decks", "cards", "refresh_tokens", "memberships", "organizations", "users"} {
		if err := db.Exec("TRUNCATE " + table + " CASCADE").Error; err != nil {
			t.Fatalf("truncar %s: %v", table, err)
		}
	}

	return db
}

// gameIDByCode consulta el ID real de un juego seed para los tests.
func gameIDByCode(t *testing.T, db *gorm.DB, code string) string {
	t.Helper()
	var id string
	if err := db.Raw("SELECT id FROM games WHERE code = ?", code).Scan(&id).Error; err != nil {
		t.Fatalf("consultar juego seed %q: %v", code, err)
	}
	if id == "" {
		t.Fatalf("el seed de juegos no incluye %q", code)
	}
	return id
}

func sampleUser() *domain.User {
	now := time.Now().UTC()
	return &domain.User{
		Username:     "romina",
		Email:        "romina@example.com",
		PasswordHash: "hash",
		CreatedAt:    now,
		UpdatedAt:    now,
	}
}

func TestIntegration_UserRepository_CRUD(t *testing.T) {
	db := setupTestDB(t)
	repo := NewUserRepository(db)
	ctx := context.Background()

	user := sampleUser()
	if err := repo.Create(ctx, user); err != nil {
		t.Fatalf("Create: %v", err)
	}
	if user.ID == "" {
		t.Fatal("Create no asignó el ID")
	}

	found, err := repo.FindByUsername(ctx, "romina")
	if err != nil {
		t.Fatalf("FindByUsername: %v", err)
	}
	if found.ID != user.ID {
		t.Errorf("FindByUsername devolvió otro usuario")
	}

	// Username duplicado → ErrConflict.
	duplicate := sampleUser()
	duplicate.Email = "otra@example.com"
	if err := repo.Create(ctx, duplicate); !errors.Is(err, domain.ErrConflict) {
		t.Fatalf("duplicado: esperaba ErrConflict, obtuve %v", err)
	}

	found.Email = "nuevo@example.com"
	found.UpdatedAt = time.Now().UTC()
	if err := repo.Update(ctx, found); err != nil {
		t.Fatalf("Update: %v", err)
	}

	byID, err := repo.FindByID(ctx, user.ID)
	if err != nil {
		t.Fatalf("FindByID: %v", err)
	}
	if byID.Email != "nuevo@example.com" {
		t.Errorf("Update no persistió el cambio")
	}

	if err := repo.Delete(ctx, user.ID); err != nil {
		t.Fatalf("Delete: %v", err)
	}
	if _, err := repo.FindByID(ctx, user.ID); !errors.Is(err, domain.ErrNotFound) {
		t.Errorf("tras Delete esperaba ErrNotFound, obtuve %v", err)
	}

	// ID malformado → ErrInvalidID (nunca error interno).
	if _, err := repo.FindByID(ctx, "no-es-uuid"); !errors.Is(err, domain.ErrInvalidID) {
		t.Errorf("ID malformado: esperaba ErrInvalidID, obtuve %v", err)
	}
}

func TestIntegration_RefreshTokenRepository(t *testing.T) {
	db := setupTestDB(t)
	ctx := context.Background()

	userRepo := NewUserRepository(db)
	repo := NewRefreshTokenRepository(db)

	user := sampleUser()
	if err := userRepo.Create(ctx, user); err != nil {
		t.Fatalf("preparar usuario: %v", err)
	}

	token := &domain.RefreshToken{
		UserID:    user.ID,
		TokenHash: "aabbccddeeff00112233445566778899aabbccddeeff00112233445566778899",
		ExpiresAt: time.Now().UTC().Add(24 * time.Hour),
		CreatedAt: time.Now().UTC(),
	}
	if err := repo.Save(ctx, token); err != nil {
		t.Fatalf("Save: %v", err)
	}

	found, err := repo.FindByHash(ctx, token.TokenHash)
	if err != nil {
		t.Fatalf("FindByHash: %v", err)
	}
	if found.UserID != user.ID || found.IsValid(time.Now()) != true {
		t.Errorf("token guardado inconsistente: %+v", found)
	}

	if err := repo.Revoke(ctx, found.ID); err != nil {
		t.Fatalf("Revoke: %v", err)
	}
	revoked, _ := repo.FindByHash(ctx, token.TokenHash)
	if revoked.RevokedAt == nil {
		t.Error("Revoke no persistió revoked_at")
	}

	// RevokeAllForUser no revoca nada si ya no hay vigentes.
	if err := repo.RevokeAllForUser(ctx, user.ID); err != nil {
		t.Fatalf("RevokeAllForUser: %v", err)
	}
}

func TestIntegration_GameRepository(t *testing.T) {
	db := setupTestDB(t)
	repo := NewGameRepository(db)
	ctx := context.Background()

	// Los seeds de la migración 00001 incluyen los 4 juegos.
	games, err := repo.FindActive(ctx)
	if err != nil {
		t.Fatalf("FindActive: %v", err)
	}
	if len(games) < 4 {
		t.Errorf("juegos activos = %d, esperaba al menos los 4 seeds", len(games))
	}

	mtg, err := repo.FindByCode(ctx, "mtg")
	if err != nil {
		t.Fatalf("FindByCode(mtg): %v", err)
	}
	if len(mtg.Formats) == 0 {
		t.Error("mtg vino sin formatos")
	}
	commander := mtg.FormatByCode("commander")
	if commander == nil {
		t.Fatal("mtg sin formato commander")
	}
	if !commander.RequiresLeader || commander.MinSize != 100 {
		t.Errorf("commander = %+v, esperaba requires_leader y min 100", commander)
	}

	if _, err := repo.FindByCode(ctx, "lorcana"); !errors.Is(err, domain.ErrNotFound) {
		t.Errorf("juego inexistente: esperaba ErrNotFound, obtuve %v", err)
	}
}

func TestIntegration_CardRepository_UpsertYBusqueda(t *testing.T) {
	db := setupTestDB(t)
	ctx := context.Background()
	repo := NewCardRepository(db)
	gameID := gameIDByCode(t, db, "mtg")

	attrs := map[string]any{"colors": []any{"Blue"}, "mana_value": 3.0, "type": "Creature"}
	card := &domain.Card{
		GameID: gameID, Name: "Atraxa", SerialSet: "CMD-1",
		Image: "https://example.com/a.png", Attributes: attrs,
	}

	first, err := repo.Upsert(ctx, card)
	if err != nil {
		t.Fatalf("Upsert: %v", err)
	}
	if first.ID == "" {
		t.Fatal("Upsert no asignó ID")
	}

	// Mismo upsert → devuelve el registro existente sin duplicar.
	again, err := repo.Upsert(ctx, card)
	if err != nil {
		t.Fatalf("Upsert repetido: %v", err)
	}
	if again.ID != first.ID {
		t.Errorf("el upsert duplicó la carta: %s != %s", again.ID, first.ID)
	}

	// Roundtrip del JSONB attributes.
	found, err := repo.FindByID(ctx, first.ID)
	if err != nil {
		t.Fatalf("FindByID: %v", err)
	}
	if found.Attributes["type"] != "Creature" {
		t.Errorf("attributes = %+v, esperaba type=Creature", found.Attributes)
	}
	if mv, ok := found.Attributes["mana_value"].(float64); !ok || mv != 3.0 {
		t.Errorf("mana_value = %v, esperaba 3.0", found.Attributes["mana_value"])
	}

	// Búsqueda por subcadena + filtro de juego.
	if _, err := repo.Upsert(ctx, &domain.Card{
		GameID: gameID, Name: "Lightning Bolt", SerialSet: "LEA-1",
		Image:      "https://example.com/lb.png",
		Attributes: map[string]any{"colors": []any{"Red"}, "mana_value": 1.0, "type": "Instant"},
	}); err != nil {
		t.Fatalf("Upsert Bolt: %v", err)
	}

	hits, total, err := repo.Find(ctx, ports.CardFilter{GameCode: "mtg", Query: "lightning"})
	if err != nil {
		t.Fatalf("Find: %v", err)
	}
	if total != 1 || len(hits) != 1 || hits[0].Name != "Lightning Bolt" {
		t.Errorf("Find devolvió %d/%d, esperaba 1 Lightning Bolt", len(hits), total)
	}

	// Caracteres comodín escapados: "%a%" no debe casar con Atraxa por el %.
	hits, _, err = repo.Find(ctx, ports.CardFilter{GameCode: "mtg", Query: "%a%"})
	if err != nil {
		t.Fatalf("Find con comodines: %v", err)
	}
	for _, h := range hits {
		if h.Name == "Atraxa" {
			t.Error("el % de la búsqueda no quedó escapado")
		}
	}

	if _, err := repo.FindByID(ctx, "00000000-0000-0000-0000-000000000000"); !errors.Is(err, domain.ErrNotFound) {
		t.Errorf("carta inexistente: esperaba ErrNotFound, obtuve %v", err)
	}
}

func TestIntegration_DeckRepository_CRUD(t *testing.T) {
	db := setupTestDB(t)
	ctx := context.Background()

	userRepo := NewUserRepository(db)
	cardRepo := NewCardRepository(db)
	repo := NewDeckRepository(db)

	user := sampleUser()
	if err := userRepo.Create(ctx, user); err != nil {
		t.Fatalf("preparar usuario: %v", err)
	}
	gameID := gameIDByCode(t, db, "mtg")

	// El caso de uso resuelve las cartas al catálogo antes de persistir.
	leader, err := cardRepo.Upsert(ctx, &domain.Card{
		GameID: gameID, Name: "Atraxa", SerialSet: "CMD-1", Image: "https://example.com/a.png",
		Attributes: map[string]any{"colors": []any{"White", "Blue"}, "mana_value": 4.0, "type": "Creature"},
	})
	if err != nil {
		t.Fatalf("Upsert líder: %v", err)
	}
	solRing, err := cardRepo.Upsert(ctx, &domain.Card{
		GameID: gameID, Name: "Sol Ring", SerialSet: "CMD-1", Image: "https://example.com/s.png",
		Attributes: map[string]any{"colors": []any{"Colorless"}, "mana_value": 2.0, "type": "Artifact"},
	})
	if err != nil {
		t.Fatalf("Upsert Sol Ring: %v", err)
	}

	now := time.Now().UTC()
	deck := &domain.Deck{
		OwnerID:    user.ID,
		GameID:     gameID,
		GameCode:   "mtg",
		Name:       "Mazo de Pruebas",
		DeckSize:   100,
		Format:     "commander",
		LeaderCard: leader,
		Cards: []domain.CardEntry{
			{Card: *solRing, Quantity: 1},
		},
		CreatedAt: now,
		UpdatedAt: now,
	}

	if err := repo.Create(ctx, deck); err != nil {
		t.Fatalf("Create: %v", err)
	}
	if deck.ID == "" {
		t.Fatal("Create no asignó el ID")
	}

	// Roundtrip con el esquema normalizado (FKs + join con el catálogo).
	found, err := repo.FindByID(ctx, deck.ID)
	if err != nil {
		t.Fatalf("FindByID: %v", err)
	}
	if found.GameCode != "mtg" {
		t.Errorf("game = %q, esperaba mtg (JOIN con games)", found.GameCode)
	}
	if found.LeaderCard == nil || found.LeaderCard.Name != "Atraxa" || found.LeaderCard.ID != leader.ID {
		t.Errorf("leader = %+v, esperaba Atraxa resuelta del catálogo", found.LeaderCard)
	}
	if len(found.Cards) != 1 || found.Cards[0].Card.Name != "Sol Ring" ||
		found.Cards[0].Card.ID != solRing.ID || found.Cards[0].Quantity != 1 {
		t.Errorf("cards = %+v, esperaba Sol Ring x1 desde deck_cards", found.Cards)
	}

	// Filtro por líder (JOIN contra catálogo, ya no JSONB).
	byLeader, err := repo.Find(ctx, ports.DeckFilter{LeaderName: "Atraxa"})
	if err != nil {
		t.Fatalf("Find por leader: %v", err)
	}
	if len(byLeader) != 1 {
		t.Errorf("Find por leader devolvió %d, esperaba 1", len(byLeader))
	}
	if byLeader[0].LeaderCard == nil || byLeader[0].LeaderCard.ID != leader.ID {
		t.Errorf("el listado debe incluir el líder resuelto: %+v", byLeader[0].LeaderCard)
	}

	// Filtro por juego.
	byGame, err := repo.Find(ctx, ports.DeckFilter{GameCode: "mtg"})
	if err != nil {
		t.Fatalf("Find por game: %v", err)
	}
	if len(byGame) != 1 {
		t.Errorf("Find por game devolvió %d, esperaba 1", len(byGame))
	}
	if byOther, _ := repo.Find(ctx, ports.DeckFilter{GameCode: "pokemon"}); len(byOther) != 0 {
		t.Errorf("Find por pokemon devolvió %d, esperaba 0", len(byOther))
	}

	// Filtro por formato inexistente.
	none, err := repo.Find(ctx, ports.DeckFilter{Format: "pauper"})
	if err != nil {
		t.Fatalf("Find: %v", err)
	}
	if len(none) != 0 {
		t.Errorf("Find por formato inesperado devolvió %d", len(none))
	}

	// Update: renombramos y cambiamos las cartas del mazo.
	found.Name = "Mazo Renombrado"
	bolt, err := cardRepo.Upsert(ctx, &domain.Card{
		GameID: gameID, Name: "Lightning Bolt", SerialSet: "LEA-1",
		Image:      "https://example.com/lb.png",
		Attributes: map[string]any{"colors": []any{"Red"}, "mana_value": 1.0, "type": "Instant"},
	})
	if err != nil {
		t.Fatalf("Upsert Bolt: %v", err)
	}
	found.Cards = append(found.Cards, domain.CardEntry{Card: *bolt, Quantity: 4})
	found.UpdatedAt = time.Now().UTC()
	if err := repo.Update(ctx, found); err != nil {
		t.Fatalf("Update: %v", err)
	}

	updated, _ := repo.FindByID(ctx, deck.ID)
	if updated.Name != "Mazo Renombrado" {
		t.Error("Update no persistió el nombre")
	}
	if len(updated.Cards) != 2 {
		t.Errorf("cartas tras update = %d, esperaba 2", len(updated.Cards))
	}

	if err := repo.Delete(ctx, deck.ID); err != nil {
		t.Fatalf("Delete: %v", err)
	}
	if _, err := repo.FindByID(ctx, deck.ID); !errors.Is(err, domain.ErrNotFound) {
		t.Errorf("tras Delete esperaba ErrNotFound, obtuve %v", err)
	}
	// deck_cards se elimina en cascada.
	var leftover int64
	db.Raw("SELECT COUNT(*) FROM deck_cards WHERE deck_id = ?", deck.ID).Scan(&leftover)
	if leftover != 0 {
		t.Errorf("deck_cards quedó con %d huérfanos tras el delete", leftover)
	}
}
