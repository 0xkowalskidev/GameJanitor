package handlers

import (
	"encoding/json"
	"log/slog"
	"net/http"
	"strconv"

	"github.com/0xkowalskidev/gamejanitor/internal/models"
	"github.com/0xkowalskidev/gamejanitor/internal/service"
	"github.com/go-chi/chi/v5"
)

type PageGameHandlers struct {
	gameSvc       *service.GameService
	gameserverSvc *service.GameserverService
	renderer      *Renderer
	log           *slog.Logger
}

func NewPageGameHandlers(gameSvc *service.GameService, gameserverSvc *service.GameserverService, renderer *Renderer, log *slog.Logger) *PageGameHandlers {
	return &PageGameHandlers{gameSvc: gameSvc, gameserverSvc: gameserverSvc, renderer: renderer, log: log}
}

func (h *PageGameHandlers) List(w http.ResponseWriter, r *http.Request) {
	games, err := h.gameSvc.ListGames()
	if err != nil {
		h.log.Error("listing games", "error", err)
		http.Error(w, "Failed to load games", http.StatusInternalServerError)
		return
	}
	if games == nil {
		games = []models.Game{}
	}

	h.renderer.Render(w, r, "games/list", map[string]any{
		"Games": games,
	})
}

func (h *PageGameHandlers) Detail(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	game, err := h.gameSvc.GetGame(id)
	if err != nil {
		h.log.Error("getting game", "id", id, "error", err)
		http.Error(w, "Failed to load game", http.StatusInternalServerError)
		return
	}
	if game == nil {
		http.Error(w, "Game not found", http.StatusNotFound)
		return
	}

	// Find gameservers using this game
	gameservers, err := h.gameserverSvc.ListGameservers(models.GameserverFilter{GameID: &id})
	if err != nil {
		h.log.Error("listing gameservers for game", "game_id", id, "error", err)
		gameservers = []models.Gameserver{}
	}

	h.renderer.Render(w, r, "games/detail", map[string]any{
		"Game":        game,
		"Gameservers": gameservers,
	})
}

func (h *PageGameHandlers) New(w http.ResponseWriter, r *http.Request) {
	h.renderer.Render(w, r, "games/new", map[string]any{})
}

func (h *PageGameHandlers) Create(w http.ResponseWriter, r *http.Request) {
	if err := r.ParseForm(); err != nil {
		http.Error(w, "Invalid form data", http.StatusBadRequest)
		return
	}

	id := r.FormValue("id")
	name := r.FormValue("name")
	image := r.FormValue("image")
	if id == "" || name == "" || image == "" {
		http.Error(w, "ID, name, and image are required", http.StatusBadRequest)
		return
	}

	game := h.parseGameForm(r, id)

	if err := h.gameSvc.CreateGame(game); err != nil {
		h.log.Error("creating game from web form", "id", id, "error", err)
		http.Error(w, "Failed to create game: "+err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("HX-Redirect", "/games/"+id)
	http.Redirect(w, r, "/games/"+id, http.StatusSeeOther)
}

func (h *PageGameHandlers) Edit(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	game, err := h.gameSvc.GetGame(id)
	if err != nil {
		h.log.Error("getting game for edit", "id", id, "error", err)
		http.Error(w, "Failed to load game", http.StatusInternalServerError)
		return
	}
	if game == nil {
		http.Error(w, "Game not found", http.StatusNotFound)
		return
	}

	defaultPortsJSON := "[]"
	if len(game.DefaultPorts) > 0 {
		defaultPortsJSON = string(game.DefaultPorts)
	}
	defaultEnvJSON := "[]"
	if len(game.DefaultEnv) > 0 {
		defaultEnvJSON = string(game.DefaultEnv)
	}
	disabledCapsJSON := "[]"
	if len(game.DisabledCapabilities) > 0 {
		disabledCapsJSON = string(game.DisabledCapabilities)
	}

	h.renderer.Render(w, r, "games/edit", map[string]any{
		"Game":             game,
		"DefaultPortsJSON": defaultPortsJSON,
		"DefaultEnvJSON":   defaultEnvJSON,
		"DisabledCapsJSON": disabledCapsJSON,
	})
}

func (h *PageGameHandlers) Update(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	if err := r.ParseForm(); err != nil {
		http.Error(w, "Invalid form data", http.StatusBadRequest)
		return
	}

	existing, err := h.gameSvc.GetGame(id)
	if err != nil {
		h.log.Error("getting game for update", "id", id, "error", err)
		http.Error(w, "Failed to load game", http.StatusInternalServerError)
		return
	}
	if existing == nil {
		http.Error(w, "Game not found", http.StatusNotFound)
		return
	}

	game := h.parseGameForm(r, id)
	game.CreatedAt = existing.CreatedAt

	if err := h.gameSvc.UpdateGame(game); err != nil {
		h.log.Error("updating game from web form", "id", id, "error", err)
		http.Error(w, "Failed to update game: "+err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("HX-Redirect", "/games/"+id)
	http.Redirect(w, r, "/games/"+id, http.StatusSeeOther)
}

func (h *PageGameHandlers) Delete(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	if err := h.gameSvc.DeleteGame(id); err != nil {
		h.log.Error("deleting game from web", "id", id, "error", err)
		http.Error(w, "Failed to delete game: "+err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("HX-Redirect", "/games")
	http.Redirect(w, r, "/games", http.StatusSeeOther)
}

func (h *PageGameHandlers) parseGameForm(r *http.Request, id string) *models.Game {
	minMemoryMB, _ := strconv.Atoi(r.FormValue("min_memory_mb"))
	minCPU, _ := strconv.ParseFloat(r.FormValue("min_cpu"), 64)

	var defaultPorts json.RawMessage
	if v := r.FormValue("default_ports_json"); v != "" {
		defaultPorts = json.RawMessage(v)
	} else {
		defaultPorts = json.RawMessage("[]")
	}

	var defaultEnv json.RawMessage
	if v := r.FormValue("default_env_json"); v != "" {
		defaultEnv = json.RawMessage(v)
	} else {
		defaultEnv = json.RawMessage("[]")
	}

	var disabledCaps json.RawMessage
	if v := r.FormValue("disabled_capabilities_json"); v != "" {
		disabledCaps = json.RawMessage(v)
	} else {
		disabledCaps = json.RawMessage("[]")
	}

	var gsqSlug *string
	if v := r.FormValue("gsq_game_slug"); v != "" {
		gsqSlug = &v
	}

	return &models.Game{
		ID:                   id,
		Name:                 r.FormValue("name"),
		Image:                r.FormValue("image"),
		IconPath:             r.FormValue("icon_path"),
		GridPath:             r.FormValue("grid_path"),
		HeroPath:             r.FormValue("hero_path"),
		DefaultPorts:         defaultPorts,
		DefaultEnv:           defaultEnv,
		MinMemoryMB:          minMemoryMB,
		MinCPU:               minCPU,
		GSQGameSlug:          gsqSlug,
		DisabledCapabilities: disabledCaps,
	}
}
