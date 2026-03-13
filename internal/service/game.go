package service

import (
	"database/sql"
	"fmt"
	"log/slog"
	"strings"

	"github.com/0xkowalskidev/gamejanitor/internal/models"
)

type GameService struct {
	db  *sql.DB
	log *slog.Logger
}

func NewGameService(db *sql.DB, log *slog.Logger) *GameService {
	return &GameService{db: db, log: log}
}

func (s *GameService) ListGames() ([]models.Game, error) {
	return models.ListGames(s.db)
}

func (s *GameService) GetGame(id string) (*models.Game, error) {
	return models.GetGame(s.db, id)
}

func (s *GameService) CreateGame(game *models.Game) error {
	s.log.Info("creating game", "id", game.ID, "name", game.Name)
	return models.CreateGame(s.db, game)
}

func (s *GameService) UpdateGame(game *models.Game) error {
	s.log.Info("updating game", "id", game.ID)
	err := models.UpdateGame(s.db, game)
	if err != nil && strings.Contains(err.Error(), "not found") {
		return ErrNotFound(err.Error())
	}
	return err
}

func (s *GameService) DeleteGame(id string) error {
	s.log.Info("deleting game", "id", id)
	err := models.DeleteGame(s.db, id)
	if err != nil {
		if strings.Contains(err.Error(), "still reference") {
			return ErrConflict(err.Error())
		}
		if strings.Contains(err.Error(), "not found") {
			return ErrNotFound(err.Error())
		}
		return fmt.Errorf("deleting game %s: %w", id, err)
	}
	return nil
}
