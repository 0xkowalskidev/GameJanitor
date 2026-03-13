package handlers

import (
	"encoding/json"
	"log/slog"
	"net/http"

	"github.com/0xkowalskidev/gamejanitor/internal/models"
	"github.com/0xkowalskidev/gamejanitor/internal/service"
	"github.com/go-chi/chi/v5"
)

type GameHandlers struct {
	svc *service.GameService
	log *slog.Logger
}

func NewGameHandlers(svc *service.GameService, log *slog.Logger) *GameHandlers {
	return &GameHandlers{svc: svc, log: log}
}

func (h *GameHandlers) List(w http.ResponseWriter, r *http.Request) {
	games, err := h.svc.ListGames()
	if err != nil {
		h.log.Error("listing games", "error", err)
		respondError(w, http.StatusInternalServerError, err.Error())
		return
	}
	if games == nil {
		games = []models.Game{}
	}
	respondOK(w, games)
}

func (h *GameHandlers) Get(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	game, err := h.svc.GetGame(id)
	if err != nil {
		h.log.Error("getting game", "id", id, "error", err)
		respondError(w, http.StatusInternalServerError, err.Error())
		return
	}
	if game == nil {
		respondError(w, http.StatusNotFound, "game "+id+" not found")
		return
	}
	respondOK(w, game)
}

func (h *GameHandlers) Create(w http.ResponseWriter, r *http.Request) {
	var game models.Game
	if err := json.NewDecoder(r.Body).Decode(&game); err != nil {
		respondError(w, http.StatusBadRequest, "invalid JSON: "+err.Error())
		return
	}

	if game.ID == "" || game.Name == "" || game.Image == "" {
		respondError(w, http.StatusBadRequest, "id, name, and image are required")
		return
	}
	if !validGameID.MatchString(game.ID) {
		respondError(w, http.StatusBadRequest, "game ID must contain only lowercase letters, numbers, and hyphens")
		return
	}

	if err := h.svc.CreateGame(&game); err != nil {
		h.log.Error("creating game", "id", game.ID, "error", err)
		respondError(w, serviceErrorStatus(err), err.Error())
		return
	}
	respondCreated(w, game)
}

func (h *GameHandlers) Update(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	var game models.Game
	if err := json.NewDecoder(r.Body).Decode(&game); err != nil {
		respondError(w, http.StatusBadRequest, "invalid JSON: "+err.Error())
		return
	}
	game.ID = id

	if err := h.svc.UpdateGame(&game); err != nil {
		h.log.Error("updating game", "id", id, "error", err)
		respondError(w, serviceErrorStatus(err), err.Error())
		return
	}
	respondOK(w, game)
}

func (h *GameHandlers) Delete(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	if err := h.svc.DeleteGame(id); err != nil {
		h.log.Error("deleting game", "id", id, "error", err)
		respondError(w, serviceErrorStatus(err), err.Error())
		return
	}
	respondNoContent(w)
}
