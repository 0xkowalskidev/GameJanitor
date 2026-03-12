package handlers

import (
	"encoding/json"
	"io"
	"log/slog"
	"net/http"

	"github.com/0xkowalskidev/gamejanitor/internal/service"
	"github.com/go-chi/chi/v5"
)

type PageFileHandlers struct {
	fileSvc       *service.FileService
	gameSvc       *service.GameService
	gameserverSvc *service.GameserverService
	renderer      *Renderer
	log           *slog.Logger
}

func NewPageFileHandlers(fileSvc *service.FileService, gameSvc *service.GameService, gameserverSvc *service.GameserverService, renderer *Renderer, log *slog.Logger) *PageFileHandlers {
	return &PageFileHandlers{fileSvc: fileSvc, gameSvc: gameSvc, gameserverSvc: gameserverSvc, renderer: renderer, log: log}
}

func (h *PageFileHandlers) List(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	dirPath := r.URL.Query().Get("path")
	if dirPath == "" {
		dirPath = "/data"
	}

	gs, err := h.gameserverSvc.GetGameserver(id)
	if err != nil {
		h.log.Error("getting gameserver for files", "id", id, "error", err)
		http.Error(w, "Failed to load gameserver", http.StatusInternalServerError)
		return
	}
	if gs == nil {
		http.Error(w, "Gameserver not found", http.StatusNotFound)
		return
	}

	game, err := h.gameSvc.GetGame(gs.GameID)
	if err != nil {
		h.log.Error("getting game for files", "game_id", gs.GameID, "error", err)
		http.Error(w, "Failed to load game", http.StatusInternalServerError)
		return
	}

	entries, err := h.fileSvc.ListDirectory(r.Context(), id, dirPath)
	if err != nil {
		h.log.Error("listing directory", "gameserver_id", id, "path", dirPath, "error", err)
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	entriesJSON, err := json.Marshal(entries)
	if err != nil {
		h.log.Error("marshaling entries", "error", err)
		http.Error(w, "Failed to encode entries", http.StatusInternalServerError)
		return
	}

	h.renderer.Render(w, r, "gameservers/files", map[string]any{
		"Gameserver":  gs,
		"Game":        game,
		"Path":        dirPath,
		"Entries":     entries,
		"EntriesJSON": string(entriesJSON),
	})
}

func (h *PageFileHandlers) ListJSON(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	dirPath := r.URL.Query().Get("path")
	if dirPath == "" {
		dirPath = "/data"
	}

	entries, err := h.fileSvc.ListDirectory(r.Context(), id, dirPath)
	if err != nil {
		h.log.Error("listing directory", "gameserver_id", id, "path", dirPath, "error", err)
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(entries)
}

func (h *PageFileHandlers) ReadFile(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	filePath := r.URL.Query().Get("path")
	if filePath == "" {
		http.Error(w, "path parameter is required", http.StatusBadRequest)
		return
	}

	content, err := h.fileSvc.ReadFile(r.Context(), id, filePath)
	if err != nil {
		h.log.Error("reading file", "gameserver_id", id, "path", filePath, "error", err)
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	w.Header().Set("Content-Type", "text/plain; charset=utf-8")
	w.Write(content)
}

func (h *PageFileHandlers) WriteFile(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	filePath := r.URL.Query().Get("path")
	if filePath == "" {
		http.Error(w, "path parameter is required", http.StatusBadRequest)
		return
	}

	content, err := io.ReadAll(io.LimitReader(r.Body, 10*1024*1024))
	if err != nil {
		http.Error(w, "Failed to read request body", http.StatusBadRequest)
		return
	}

	if err := h.fileSvc.WriteFile(r.Context(), id, filePath, content); err != nil {
		h.log.Error("writing file", "gameserver_id", id, "path", filePath, "error", err)
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

func (h *PageFileHandlers) DeletePath(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	targetPath := r.URL.Query().Get("path")
	if targetPath == "" {
		http.Error(w, "path parameter is required", http.StatusBadRequest)
		return
	}

	if err := h.fileSvc.DeletePath(r.Context(), id, targetPath); err != nil {
		h.log.Error("deleting path", "gameserver_id", id, "path", targetPath, "error", err)
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

func (h *PageFileHandlers) CreateDirectory(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	dirPath := r.URL.Query().Get("path")
	if dirPath == "" {
		http.Error(w, "path parameter is required", http.StatusBadRequest)
		return
	}

	if err := h.fileSvc.CreateDirectory(r.Context(), id, dirPath); err != nil {
		h.log.Error("creating directory", "gameserver_id", id, "path", dirPath, "error", err)
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	w.WriteHeader(http.StatusNoContent)
}
