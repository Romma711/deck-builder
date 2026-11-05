package postgres

import (
	"context"
	"errors"
	"fmt"

	"github.com/Romma711/deck-builder/internal/domain"
	"github.com/Romma711/deck-builder/internal/ports"
	"gorm.io/gorm"
)

// GameRepository adapta ports.GameRepository a PostgreSQL (games + formats).
type GameRepository struct {
	db *gorm.DB
}

var _ ports.GameRepository = (*GameRepository)(nil)

func NewGameRepository(db *gorm.DB) *GameRepository {
	return &GameRepository{db: db}
}

func (r *GameRepository) FindByCode(ctx context.Context, code string) (*domain.Game, error) {
	var model GameModel
	err := r.db.WithContext(ctx).Where("code = ?", code).First(&model).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, domain.ErrNotFound
		}
		return nil, fmt.Errorf("buscar juego: %w", err)
	}
	if !model.Active {
		return nil, domain.ErrNotFound
	}

	game, err := r.toDomain(model)
	if err != nil {
		return nil, err
	}
	if err := r.loadFormats(ctx, game); err != nil {
		return nil, err
	}
	return game, nil
}

func (r *GameRepository) FindActive(ctx context.Context) ([]domain.Game, error) {
	var models []GameModel
	if err := r.db.WithContext(ctx).Where("active = ?", true).Order("code").Find(&models).Error; err != nil {
		return nil, fmt.Errorf("listar juegos: %w", err)
	}

	games := make([]domain.Game, 0, len(models))
	for _, model := range models {
		game, err := r.toDomain(model)
		if err != nil {
			return nil, err
		}
		games = append(games, *game)
	}

	// Cargamos todos los formatos de una sola query y los agrupamos.
	var formatModels []FormatModel
	gameIDs := make([]string, 0, len(games))
	for i := range games {
		gameIDs = append(gameIDs, games[i].ID)
	}
	if err := r.db.WithContext(ctx).
		Where("game_id IN ?", gameIDs).
		Order("code").Find(&formatModels).Error; err != nil {
		return nil, fmt.Errorf("listar formatos: %w", err)
	}

	formatsByGame := map[string][]domain.Format{}
	for _, fm := range formatModels {
		formatsByGame[fm.GameID.String()] = append(
			formatsByGame[fm.GameID.String()], formatToDomain(fm))
	}
	for i := range games {
		games[i].Formats = formatsByGame[games[i].ID]
	}

	return games, nil
}

func (r *GameRepository) loadFormats(ctx context.Context, game *domain.Game) error {
	var formatModels []FormatModel
	if err := r.db.WithContext(ctx).Where("game_id = ?", game.ID).
		Order("code").Find(&formatModels).Error; err != nil {
		return fmt.Errorf("listar formatos: %w", err)
	}
	game.Formats = make([]domain.Format, 0, len(formatModels))
	for _, fm := range formatModels {
		game.Formats = append(game.Formats, formatToDomain(fm))
	}
	return nil
}

func (r *GameRepository) toDomain(model GameModel) (*domain.Game, error) {
	return &domain.Game{
		ID:     model.ID.String(),
		Code:   model.Code,
		Name:   model.Name,
		Active: model.Active,
	}, nil
}

func formatToDomain(model FormatModel) domain.Format {
	return domain.Format{
		ID:             model.ID.String(),
		GameID:         model.GameID.String(),
		Code:           model.Code,
		Name:           model.Name,
		DeckSize:       model.DeckSize,
		MinSize:        model.MinSize,
		MaxSize:        model.MaxSize,
		RequiresLeader: model.RequiresLeader,
	}
}
