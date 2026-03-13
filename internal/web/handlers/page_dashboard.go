package handlers

import (
	"encoding/json"
	"fmt"
	"log/slog"
	"net/http"

	"github.com/0xkowalskidev/gamejanitor/internal/models"
	"github.com/0xkowalskidev/gamejanitor/internal/service"
)

type PageDashboardHandlers struct {
	gameSvc       *service.GameService
	gameserverSvc *service.GameserverService
	querySvc      *service.QueryService
	renderer      *Renderer
	log           *slog.Logger
}

func NewPageDashboardHandlers(gameSvc *service.GameService, gameserverSvc *service.GameserverService, querySvc *service.QueryService, renderer *Renderer, log *slog.Logger) *PageDashboardHandlers {
	return &PageDashboardHandlers{gameSvc: gameSvc, gameserverSvc: gameserverSvc, querySvc: querySvc, renderer: renderer, log: log}
}

type gameserverView struct {
	ID            string
	Name          string
	GameID        string
	GameName      string
	GridPath      string
	HeroPath      string
	IconPath      string
	GamePort      string
	Status        string
	PlayersOnline int
	MaxPlayers    int
	HasQueryData  bool
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

	var activeViews, stoppedViews []gameserverView
	for _, gs := range gameservers {
		game := gameLookup[gs.GameID]
		v := gameserverView{
			ID:       gs.ID,
			Name:     gs.Name,
			GameID:   gs.GameID,
			GameName: game.Name,
			GridPath: game.GridPath,
			HeroPath: game.HeroPath,
			IconPath: game.IconPath,
			Status:   gs.Status,
			GamePort: firstGamePort(gs.Ports),
		}
		if qd := h.querySvc.GetQueryData(gs.ID); qd != nil {
			v.PlayersOnline = qd.PlayersOnline
			v.MaxPlayers = qd.MaxPlayers
			v.HasQueryData = true
		}
		if gs.Status == "stopped" {
			stoppedViews = append(stoppedViews, v)
		} else {
			activeViews = append(activeViews, v)
		}
	}

	h.renderer.Render(w, r, "dashboard", map[string]any{
		"ActiveGameservers":  activeViews,
		"StoppedGameservers": stoppedViews,
		"HasGameservers":     len(gameservers) > 0,
		"Games":              games,
		"RunningCount":       len(activeViews),
		"StoppedCount":       len(stoppedViews),
		"TotalCount":         len(gameservers),
	})
}

// firstGamePort extracts the first game port number from a gameserver's port config.
func firstGamePort(portsJSON json.RawMessage) string {
	if len(portsJSON) == 0 {
		return ""
	}
	var ports []struct {
		HostPort int    `json:"host_port"`
		Name     string `json:"name"`
	}
	if err := json.Unmarshal(portsJSON, &ports); err != nil || len(ports) == 0 {
		return ""
	}
	for _, p := range ports {
		if p.Name == "game" {
			return fmt.Sprintf("%d", p.HostPort)
		}
	}
	return fmt.Sprintf("%d", ports[0].HostPort)
}
