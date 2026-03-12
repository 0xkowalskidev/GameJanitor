package handlers

import (
	"log/slog"
	"net/http"

	"github.com/0xkowalskidev/gamejanitor/internal/models"
	"github.com/0xkowalskidev/gamejanitor/internal/service"
)

type PageDashboardHandlers struct {
	gameSvc       *service.GameService
	gameserverSvc *service.GameserverService
	renderer      *Renderer
	log           *slog.Logger
}

func NewPageDashboardHandlers(gameSvc *service.GameService, gameserverSvc *service.GameserverService, renderer *Renderer, log *slog.Logger) *PageDashboardHandlers {
	return &PageDashboardHandlers{gameSvc: gameSvc, gameserverSvc: gameserverSvc, renderer: renderer, log: log}
}

type gameserverView struct {
	ID       string
	Name     string
	GameID   string
	GameName string
	GridPath string
	HeroPath string
	Status   string
}

func (h *PageDashboardHandlers) Dashboard(w http.ResponseWriter, r *http.Request) {
	gameservers, err := h.gameserverSvc.ListGameservers(models.GameserverFilter{})
	if err != nil {
		h.log.Error("listing gameservers for dashboard", "error", err)
		http.Error(w, "Failed to load dashboard", http.StatusInternalServerError)
		return
	}

	// Build game lookup
	games, err := h.gameSvc.ListGames()
	if err != nil {
		h.log.Error("listing games for dashboard", "error", err)
		http.Error(w, "Failed to load dashboard", http.StatusInternalServerError)
		return
	}
	gameLookup := make(map[string]models.Game, len(games))
	for _, g := range games {
		gameLookup[g.ID] = g
	}

	views := make([]gameserverView, len(gameservers))
	for i, gs := range gameservers {
		game := gameLookup[gs.GameID]
		views[i] = gameserverView{
			ID:       gs.ID,
			Name:     gs.Name,
			GameID:   gs.GameID,
			GameName: game.Name,
			GridPath: game.GridPath,
			HeroPath: game.HeroPath,
			Status:   gs.Status,
		}
	}

	h.renderer.Render(w, r, "dashboard", map[string]any{
		"Gameservers": views,
		"Games":       games,
	})
}
