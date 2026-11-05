package deck

import (
	"context"
	"errors"
	"fmt"
	"testing"
	"time"

	"github.com/Romma711/deck-builder/internal/app/gamespec"
	"github.com/Romma711/deck-builder/internal/domain"
	"github.com/Romma711/deck-builder/internal/ports"
	"github.com/google/uuid"
)

// --- Fakes de los puertos ---

type fakeDeckRepo struct {
	decks  map[string]*domain.Deck
	nextID int
	// lastFilter registra el filtro de la última búsqueda (para assertearlo).
	lastFilter ports.DeckFilter
}

func newFakeDeckRepo() *fakeDeckRepo {
	return &fakeDeckRepo{decks: map[string]*domain.Deck{}}
}

func (f *fakeDeckRepo) Create(_ context.Context, deck *domain.Deck) error {
	f.nextID++
	deck.ID = fmt.Sprintf("deck-%d", f.nextID)
	copy := *deck
	f.decks[deck.ID] = &copy
	return nil
}

func (f *fakeDeckRepo) FindByID(_ context.Context, id string) (*domain.Deck, error) {
	if deck, ok := f.decks[id]; ok {
		copy := *deck
		return &copy, nil
	}
	return nil, domain.ErrNotFound
}

func (f *fakeDeckRepo) Find(_ context.Context, filter ports.DeckFilter) ([]domain.Deck, error) {
	f.lastFilter = filter
	var result []domain.Deck
	for _, deck := range f.decks {
		if filter.OwnerID != "" && deck.OwnerID != filter.OwnerID {
			continue
		}
		if filter.GameCode != "" && deck.GameCode != filter.GameCode {
			continue
		}
		if filter.Format != "" && deck.Format != filter.Format {
			continue
		}
		if filter.LeaderName != "" && (deck.LeaderCard == nil || deck.LeaderCard.Name != filter.LeaderName) {
			continue
		}
		result = append(result, *deck)
	}
	return result, nil
}

func (f *fakeDeckRepo) Update(_ context.Context, deck *domain.Deck) error {
	if _, ok := f.decks[deck.ID]; !ok {
		return domain.ErrNotFound
	}
	copy := *deck
	f.decks[deck.ID] = &copy
	return nil
}

func (f *fakeDeckRepo) Delete(_ context.Context, id string) error {
	if _, ok := f.decks[id]; !ok {
		return domain.ErrNotFound
	}
	delete(f.decks, id)
	return nil
}

// fakeGameRepo sirve el catálogo de juegos con formatos (espejo de los seeds
// de la migración 00001 para los juegos que usa este test).
type fakeGameRepo struct {
	games map[string]domain.Game
}

func newFakeGameRepo() *fakeGameRepo {
	return &fakeGameRepo{games: map[string]domain.Game{
		"mtg": {
			ID: "game-mtg", Code: "mtg", Name: "Magic", Active: true,
			Formats: []domain.Format{
				{ID: "f-commander", GameID: "game-mtg", Code: "commander",
					Name: "Commander", DeckSize: 100, MinSize: 100, MaxSize: 100, RequiresLeader: true},
				{ID: "f-standard", GameID: "game-mtg", Code: "standard",
					Name: "Standard", DeckSize: 60, MinSize: 60, MaxSize: 60},
			},
		},
		"yuigoh": {
			ID: "game-yuigoh", Code: "yuigoh", Name: "Yu-Gi-Oh!", Active: true,
			Formats: []domain.Format{
				{ID: "f-master", GameID: "game-yuigoh", Code: "master",
					Name: "Master Duel", MinSize: 40, MaxSize: 60},
			},
		},
		"onepiece": {
			ID: "game-onepiece", Code: "onepiece", Name: "One Piece", Active: true,
			Formats: []domain.Format{
				{ID: "f-op-standard", GameID: "game-onepiece", Code: "standard",
					Name: "Standard", MinSize: 50, MaxSize: 50, RequiresLeader: true},
			},
		},
	}}
}

func (f *fakeGameRepo) FindByCode(_ context.Context, code string) (*domain.Game, error) {
	game, ok := f.games[code]
	if !ok {
		return nil, domain.ErrNotFound
	}
	return &game, nil
}

func (f *fakeGameRepo) FindActive(_ context.Context) ([]domain.Game, error) {
	var games []domain.Game
	for _, g := range f.games {
		games = append(games, g)
	}
	return games, nil
}

// fakeCardRepo simula el catálogo: asigna IDs únicos por identidad.
type fakeCardRepo struct {
	nextID  int
	byIdent map[string]*domain.Card
}

func newFakeCardRepo() *fakeCardRepo {
	return &fakeCardRepo{byIdent: map[string]*domain.Card{}}
}

func (f *fakeCardRepo) Upsert(_ context.Context, card *domain.Card) (*domain.Card, error) {
	ident := fmt.Sprintf("%s|%s|%s", card.GameID, card.Name, card.SerialSet)
	if existing, ok := f.byIdent[ident]; ok {
		return existing, nil
	}
	f.nextID++
	stored := *card
	stored.ID = fmt.Sprintf("card-%d", f.nextID)
	f.byIdent[ident] = &stored
	return &stored, nil
}

func (f *fakeCardRepo) FindByID(_ context.Context, id string) (*domain.Card, error) {
	for _, card := range f.byIdent {
		if card.ID == id {
			return card, nil
		}
	}
	return nil, domain.ErrNotFound
}

func (f *fakeCardRepo) Find(_ context.Context, _ ports.CardFilter) ([]domain.Card, int, error) {
	return nil, 0, nil
}

// fakeMembershipRepo implementa ports.MembershipRepository para el tenancy
// de mazos (Fase 5). Valida el formato de los IDs como hace el real.
type fakeMembershipRepo struct {
	byKey map[string]domain.Membership // "orgID|userID"
}

func newFakeMembershipRepo() *fakeMembershipRepo {
	return &fakeMembershipRepo{byKey: map[string]domain.Membership{}}
}

func membershipKey(orgID, userID string) string { return orgID + "|" + userID }

func (f *fakeMembershipRepo) Add(_ context.Context, m *domain.Membership) error {
	if _, err := uuid.Parse(m.OrgID); err != nil {
		return domain.ErrInvalidID
	}
	if _, err := uuid.Parse(m.UserID); err != nil {
		return domain.ErrInvalidID
	}
	key := membershipKey(m.OrgID, m.UserID)
	if _, ok := f.byKey[key]; ok {
		return domain.ErrConflict
	}
	f.byKey[key] = *m
	return nil
}

func (f *fakeMembershipRepo) FindByOrgAndUser(_ context.Context, orgID, userID string) (*domain.Membership, error) {
	if _, err := uuid.Parse(orgID); err != nil {
		return nil, domain.ErrInvalidID
	}
	if _, err := uuid.Parse(userID); err != nil {
		return nil, domain.ErrInvalidID
	}
	m, ok := f.byKey[membershipKey(orgID, userID)]
	if !ok {
		return nil, domain.ErrNotFound
	}
	return &m, nil
}

func (f *fakeMembershipRepo) UpdateRole(_ context.Context, orgID, userID string, role domain.Role) error {
	key := membershipKey(orgID, userID)
	m, ok := f.byKey[key]
	if !ok {
		return domain.ErrNotFound
	}
	m.Role = role
	f.byKey[key] = m
	return nil
}

func (f *fakeMembershipRepo) Remove(_ context.Context, orgID, userID string) error {
	key := membershipKey(orgID, userID)
	if _, ok := f.byKey[key]; !ok {
		return domain.ErrNotFound
	}
	delete(f.byKey, key)
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

// --- Harness ---

const ownerA = "aaaaaaaa-aaaa-aaaa-aaaa-aaaaaaaaaaaa"
const ownerB = "bbbbbbbb-bbbb-bbbb-bbbb-bbbbbbbbbbbb"

// testCardMTG crea una carta válida para el spec de MTG.
func testCardMTG(name string) *domain.Card {
	return &domain.Card{
		Name: name, SerialSet: "SET-1", Image: "https://example.com/card.png",
		Attributes: map[string]any{
			"colors": []string{"Blue"}, "mana_value": 3.0, "type": "Creature",
		},
	}
}

// testCardOP crea una carta válida para el spec de One Piece.
func testCardOP(name, cardType string) *domain.Card {
	attrs := map[string]any{"card_type": cardType}
	if cardType == "Leader" {
		attrs["life"] = 5.0
	} else {
		attrs["power"] = 5000.0
	}
	return &domain.Card{
		Name: name, SerialSet: "ST-1", Image: "https://example.com/op.png",
		Attributes: attrs,
	}
}

func validInput() DeckInput {
	return DeckInput{
		Name:     "Mazo de Pruebas",
		DeckSize: 100,
		GameCode: "mtg",
		Format:   "commander",
		Leader:   testCardMTG("Atraxa"),
		Cards: []domain.CardEntry{
			{Card: *testCardMTG("Sol Ring"), Quantity: 1},
		},
	}
}

func newTestService() (*Service, *fakeDeckRepo, *fakeCardRepo, *fakeMembershipRepo) {
	deckRepo := newFakeDeckRepo()
	cardRepo := newFakeCardRepo()
	memberships := newFakeMembershipRepo()
	specs, err := gamespec.NewRegistry(
		gamespec.MTGSpec{}, gamespec.PokemonSpec{},
		gamespec.YuGiOhSpec{}, gamespec.OnePieceSpec{},
	)
	if err != nil {
		panic(err)
	}
	svc := NewService(deckRepo, cardRepo, newFakeGameRepo(), memberships, specs)
	svc.now = func() time.Time { return time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC) }
	return svc, deckRepo, cardRepo, memberships
}

// --- Tests ---

func TestCreate_CommanderConLider(t *testing.T) {
	svc, _, _, _ := newTestService()

	deck, err := svc.Create(context.Background(), ownerA, validInput())
	if err != nil {
		t.Fatalf("Create falló: %v", err)
	}
	if deck.ID == "" {
		t.Error("el mazo quedó sin ID")
	}
	if deck.GameCode != "mtg" {
		t.Errorf("game = %q, esperaba mtg", deck.GameCode)
	}
	if deck.LeaderCard == nil || deck.LeaderCard.Name != "Atraxa" {
		t.Errorf("leader = %+v, esperaba Atraxa", deck.LeaderCard)
	}
	if deck.LeaderCard.ID == "" {
		t.Error("el líder debió resolverse al catálogo (con ID)")
	}
	if deck.Format != "commander" {
		t.Errorf("format = %q, esperaba commander", deck.Format)
	}
}

func TestCreate_ResuelveCartasAlCatalogo(t *testing.T) {
	svc, _, cardRepo, _ := newTestService()

	in := validInput()
	in.Cards = []domain.CardEntry{
		{Card: *testCardMTG("Sol Ring"), Quantity: 1},
		// Misma identidad que la anterior: debe deduplicarse sumando cantidad.
		{Card: *testCardMTG("Sol Ring"), Quantity: 2},
	}

	deck, err := svc.Create(context.Background(), ownerA, in)
	if err != nil {
		t.Fatalf("Create falló: %v", err)
	}
	if len(deck.Cards) != 1 {
		t.Fatalf("cartas = %d, esperaba 1 tras deduplicar", len(deck.Cards))
	}
	if deck.Cards[0].Quantity != 3 {
		t.Errorf("quantity = %d, esperaba 3 (1+2)", deck.Cards[0].Quantity)
	}
	if deck.Cards[0].Card.ID == "" {
		t.Error("la carta debió resolverse al catálogo (con ID)")
	}
	if len(cardRepo.byIdent) != 2 { // Atraxa + Sol Ring
		t.Errorf("catálogo = %d cartas, esperaba 2", len(cardRepo.byIdent))
	}
}

func TestCreate_JuegoNoSoportado_ValidationError(t *testing.T) {
	svc, _, _, _ := newTestService()

	in := validInput()
	in.GameCode = "lorcana"

	_, err := svc.Create(context.Background(), ownerA, in)
	var ve *domain.ValidationError
	if !errors.As(err, &ve) {
		t.Fatalf("esperaba *domain.ValidationError, obtuve %v", err)
	}
	if ve.Details["Game"] != "unsupported" {
		t.Errorf("detalles = %v, esperaba Game: unsupported", ve.Details)
	}
}

func TestCreate_LiderSinLider_ValidationError(t *testing.T) {
	svc, _, _, _ := newTestService()

	in := validInput()
	in.Leader = nil

	_, err := svc.Create(context.Background(), ownerA, in)
	var ve *domain.ValidationError
	if !errors.As(err, &ve) {
		t.Fatalf("esperaba *domain.ValidationError, obtuve %v", err)
	}
	if ve.Details["Leader"] == "" {
		t.Errorf("falta el detalle de Leader: %v", ve.Details)
	}
}

func TestCreate_FormatoInvalidoParaElJuego(t *testing.T) {
	svc, _, _, _ := newTestService()

	in := validInput()
	// "master" existe en yuigoh, no en mtg.
	in.Format = "master"

	_, err := svc.Create(context.Background(), ownerA, in)
	var ve *domain.ValidationError
	if !errors.As(err, &ve) {
		t.Fatalf("esperaba *domain.ValidationError, obtuve %v", err)
	}
	if ve.Details["Format"] == "" {
		t.Errorf("detalles = %v, esperaba detalle de Format", ve.Details)
	}
}

func TestCreate_TamanoFueraDeRango_ValidationError(t *testing.T) {
	svc, _, _, _ := newTestService()

	in := validInput()
	in.GameCode = "yuigoh"
	in.Format = "master"
	in.DeckSize = 30 // master pide 40-60
	in.Leader = nil
	in.Cards = []domain.CardEntry{
		{Card: *yuigohCard("Blue-Eyes", "Monster", 3000, 2500), Quantity: 3},
	}

	_, err := svc.Create(context.Background(), ownerA, in)
	var ve *domain.ValidationError
	if !errors.As(err, &ve) {
		t.Fatalf("esperaba *domain.ValidationError, obtuve %v", err)
	}
	if ve.Details["DeckSize"] == "" {
		t.Errorf("detalles = %v, esperaba detalle de DeckSize", ve.Details)
	}
}

func TestCreate_OnePieceLiderDebeSerTipoLeader(t *testing.T) {
	svc, _, _, _ := newTestService()

	in := DeckInput{
		Name:     "Mazo OP",
		DeckSize: 50,
		GameCode: "onepiece",
		Format:   "standard",
		// Líder válido en atributos de carta pero NO de tipo Leader.
		Leader: testCardOP("Luffy", "Character"),
		Cards: []domain.CardEntry{
			{Card: *testCardOP("Zoro", "Character"), Quantity: 4},
		},
	}

	_, err := svc.Create(context.Background(), ownerA, in)
	var ve *domain.ValidationError
	if !errors.As(err, &ve) {
		t.Fatalf("esperaba *domain.ValidationError, obtuve %v", err)
	}
	if ve.Details["leader.card_type"] == "" {
		t.Errorf("detalles = %v, esperaba leader.card_type", ve.Details)
	}
}

func TestCreate_CartaConAtributosInvalidos_AcumulaErroresPorIndice(t *testing.T) {
	svc, _, _, _ := newTestService()

	in := validInput()
	// Carta con mana_value fuera de rango y type inválido.
	bad := testCardMTG("Mala")
	bad.Attributes["mana_value"] = 99.0
	bad.Attributes["type"] = "Tribal"
	in.Cards = append(in.Cards, domain.CardEntry{Card: *bad, Quantity: 1})

	_, err := svc.Create(context.Background(), ownerA, in)
	var ve *domain.ValidationError
	if !errors.As(err, &ve) {
		t.Fatalf("esperaba *domain.ValidationError, obtuve %v", err)
	}
	if ve.Details["cards[1].mana_value"] == "" {
		t.Errorf("detalles = %v, esperaba cards[1].mana_value", ve.Details)
	}
	if ve.Details["cards[1].type"] == "" {
		t.Errorf("detalles = %v, esperaba cards[1].type", ve.Details)
	}
}

func TestCreate_MTGCopiasLimite(t *testing.T) {
	svc, _, _, _ := newTestService()

	in := validInput()
	in.DeckSize = 60
	in.Format = "standard"
	in.Leader = nil
	in.Cards = []domain.CardEntry{
		{Card: *testCardMTG("Sol Ring"), Quantity: 5}, // límite: 4
	}

	_, err := svc.Create(context.Background(), ownerA, in)
	var ve *domain.ValidationError
	if !errors.As(err, &ve) {
		t.Fatalf("esperaba *domain.ValidationError, obtuve %v", err)
	}
	if ve.Details["cards.Sol Ring"] == "" {
		t.Errorf("detalles = %v, esperaba límite de copias en cards.Sol Ring", ve.Details)
	}
}

func yuigohCard(name, cardType string, atk, def int) *domain.Card {
	attrs := map[string]any{"card_type": cardType}
	if cardType == "Monster" {
		attrs["atk"] = float64(atk)
		attrs["def"] = float64(def)
	}
	return &domain.Card{
		Name: name, SerialSet: "LOB-1", Image: "https://example.com/ygo.png",
		Attributes: attrs,
	}
}

func TestCreate_SinCartas_OSinDuenio(t *testing.T) {
	svc, _, _, _ := newTestService()

	in := validInput()
	in.Cards = nil
	if _, err := svc.Create(context.Background(), ownerA, in); err == nil {
		t.Error("un mazo sin cartas debió fallar")
	}

	if _, err := svc.Create(context.Background(), "", validInput()); !errors.Is(err, domain.ErrUnauthorized) {
		t.Errorf("sin ownerID esperaba ErrUnauthorized, obtuve %v", err)
	}
}

func TestCreate_NormalizaJuegoYFormato(t *testing.T) {
	svc, _, _, _ := newTestService()

	in := validInput()
	in.GameCode = "  MTG "
	in.Format = "  Commander "

	deck, err := svc.Create(context.Background(), ownerA, in)
	if err != nil {
		t.Fatalf("Create falló: %v", err)
	}
	if deck.GameCode != "mtg" || deck.Format != "commander" {
		t.Errorf("game/format = %q/%q, esperaba mtg/commander", deck.GameCode, deck.Format)
	}
}

func TestUpdate_NoEsDuenio_Forbidden(t *testing.T) {
	svc, _, _, _ := newTestService()
	created, _ := svc.Create(context.Background(), ownerA, validInput())

	_, err := svc.Update(context.Background(), created.ID, ownerB, validInput())
	if !errors.Is(err, domain.ErrForbidden) {
		t.Fatalf("esperaba ErrForbidden, obtuve %v", err)
	}
}

func TestUpdate_Duenio_OK(t *testing.T) {
	svc, repo, _, _ := newTestService()
	created, _ := svc.Create(context.Background(), ownerA, validInput())

	// Movemos el reloj para verificar que UpdatedAt avanza.
	svc.now = func() time.Time { return time.Date(2026, 1, 2, 0, 0, 0, 0, time.UTC) }

	in := validInput()
	in.Name = "Mazo Renombrado"

	updated, err := svc.Update(context.Background(), created.ID, ownerA, in)
	if err != nil {
		t.Fatalf("Update falló: %v", err)
	}
	if updated.Name != "Mazo Renombrado" {
		t.Errorf("name = %q", updated.Name)
	}
	if !updated.UpdatedAt.After(updated.CreatedAt) {
		t.Error("UpdatedAt debió avanzar")
	}
	if repo.decks[created.ID].Name != "Mazo Renombrado" {
		t.Error("el cambio no llegó al repositorio")
	}
}

func TestUpdate_MazoInexistente_NotFound(t *testing.T) {
	svc, _, _, _ := newTestService()

	_, err := svc.Update(context.Background(), "deck-99", ownerA, validInput())
	if !errors.Is(err, domain.ErrNotFound) {
		t.Fatalf("esperaba ErrNotFound, obtuve %v", err)
	}
}

func TestDelete_NoEsDuenio_Forbidden(t *testing.T) {
	svc, repo, _, _ := newTestService()
	created, _ := svc.Create(context.Background(), ownerA, validInput())

	if err := svc.Delete(context.Background(), created.ID, ownerB); !errors.Is(err, domain.ErrForbidden) {
		t.Fatalf("esperaba ErrForbidden, obtuve %v", err)
	}
	if _, ok := repo.decks[created.ID]; !ok {
		t.Error("el mazo no debía borrarse")
	}
}

func TestDelete_Duenio_OK(t *testing.T) {
	svc, repo, _, _ := newTestService()
	created, _ := svc.Create(context.Background(), ownerA, validInput())

	if err := svc.Delete(context.Background(), created.ID, ownerA); err != nil {
		t.Fatalf("Delete falló: %v", err)
	}
	if _, ok := repo.decks[created.ID]; ok {
		t.Error("el mazo debió eliminarse")
	}
}

func TestList_PasaFiltrosNormalizados(t *testing.T) {
	svc, repo, _, _ := newTestService()
	_, _ = svc.Create(context.Background(), ownerA, validInput())

	if _, err := svc.List(context.Background(), ports.DeckFilter{
		GameCode: " MTG ", OwnerID: ownerA, Format: " COMMANDER ", LeaderName: "Atraxa",
	}); err != nil {
		t.Fatalf("List falló: %v", err)
	}

	if repo.lastFilter.Format != "commander" || repo.lastFilter.GameCode != "mtg" {
		t.Errorf("filtro = %+v, esperaba format/game normalizados", repo.lastFilter)
	}
	if repo.lastFilter.OwnerID != ownerA || repo.lastFilter.LeaderName != "Atraxa" {
		t.Errorf("filtro = %+v, no conserva los criterios", repo.lastFilter)
	}
}

// --- Tenancy: mazos de organización (Fase 5) ---

const orgX = "cccccccc-cccc-cccc-cccc-cccccccccccc"

func TestCreate_MazoOrg_NoEsMiembro_Forbidden(t *testing.T) {
	svc, _, _, _ := newTestService()

	in := validInput()
	in.OrgID = orgX
	_, err := svc.Create(context.Background(), ownerA, in)
	if !errors.Is(err, domain.ErrForbidden) {
		t.Fatalf("err = %v, esperaba ErrForbidden (no miembro de la org)", err)
	}
}

func TestCreate_MazoOrg_EsMiembro_OK(t *testing.T) {
	svc, repo, _, memberships := newTestService()

	if err := memberships.Add(context.Background(), &domain.Membership{
		OrgID: orgX, UserID: ownerA, Role: domain.RoleMember,
	}); err != nil {
		t.Fatalf("sembrar membership: %v", err)
	}

	in := validInput()
	in.OrgID = orgX
	deck, err := svc.Create(context.Background(), ownerA, in)
	if err != nil {
		t.Fatalf("Create falló: %v", err)
	}
	if deck.OrgID != orgX {
		t.Errorf("OrgID = %q, esperaba %q", deck.OrgID, orgX)
	}
	if stored := repo.decks[deck.ID]; stored.OrgID != orgX {
		t.Errorf("persistido OrgID = %q, esperaba %q", stored.OrgID, orgX)
	}
}

func TestCreate_MazoOrg_IdBasura_InvalidID(t *testing.T) {
	svc, _, _, _ := newTestService()

	in := validInput()
	in.OrgID = "esto-no-es-uuid"
	_, err := svc.Create(context.Background(), ownerA, in)
	if !errors.Is(err, domain.ErrInvalidID) {
		t.Fatalf("err = %v, esperaba ErrInvalidID", err)
	}
}

func TestCreate_MazoPrivado_FlagPersistido(t *testing.T) {
	svc, repo, _, _ := newTestService()

	in := validInput()
	in.IsPrivate = true
	deck, err := svc.Create(context.Background(), ownerA, in)
	if err != nil {
		t.Fatalf("Create falló: %v", err)
	}
	if !deck.IsPrivate {
		t.Error("IsPrivate no se propagó al mazo")
	}
	if stored := repo.decks[deck.ID]; !stored.IsPrivate {
		t.Error("IsPrivate no se persistió")
	}
}

func TestUpdate_MazoOrg_RequiereMembresia(t *testing.T) {
	svc, _, _, memberships := newTestService()

	deck, err := svc.Create(context.Background(), ownerA, validInput())
	if err != nil {
		t.Fatalf("Create falló: %v", err)
	}

	// El dueño "se une" a la org y mueve el mazo hacia ella.
	if err := memberships.Add(context.Background(), &domain.Membership{
		OrgID: orgX, UserID: ownerA, Role: domain.RoleMember,
	}); err != nil {
		t.Fatalf("sembrar membership: %v", err)
	}
	in := validInput()
	in.OrgID = orgX
	updated, err := svc.Update(context.Background(), deck.ID, ownerA, in)
	if err != nil {
		t.Fatalf("Update falló: %v", err)
	}
	if updated.OrgID != orgX {
		t.Errorf("OrgID = %q, esperaba %q", updated.OrgID, orgX)
	}

	// Pierde la membresía: mover el mazo a la org ahora debe rechazarse
	// (la app da el 403 claro; RLS es la red de seguridad en BD).
	if err := memberships.Remove(context.Background(), orgX, ownerA); err != nil {
		t.Fatalf("remover membership: %v", err)
	}
	in2 := validInput()
	in2.OrgID = orgX
	if _, err := svc.Update(context.Background(), deck.ID, ownerA, in2); !errors.Is(err, domain.ErrForbidden) {
		t.Fatalf("err = %v, esperaba ErrForbidden tras perder la membresía", err)
	}
}
