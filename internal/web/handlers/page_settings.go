package handlers

import (
	"log/slog"
	"net/http"
	"strings"

	"github.com/0xkowalskidev/gamejanitor/internal/service"
)

type PageSettingsHandlers struct {
	settingsSvc *service.SettingsService
	renderer    *Renderer
	log         *slog.Logger
}

func NewPageSettingsHandlers(settingsSvc *service.SettingsService, renderer *Renderer, log *slog.Logger) *PageSettingsHandlers {
	return &PageSettingsHandlers{settingsSvc: settingsSvc, renderer: renderer, log: log}
}

func (h *PageSettingsHandlers) SetConnectionAddress(w http.ResponseWriter, r *http.Request) {
	if err := r.ParseForm(); err != nil {
		http.Error(w, "Invalid form data", http.StatusBadRequest)
		return
	}

	address := strings.TrimSpace(r.FormValue("connection_address"))
	if address == "" {
		http.Error(w, "Address is required", http.StatusBadRequest)
		return
	}

	if err := h.settingsSvc.SetConnectionAddress(address); err != nil {
		h.log.Error("setting connection address", "error", err)
		http.Error(w, "Failed to save connection address", http.StatusInternalServerError)
		return
	}

	// Reload the current page via HTMX
	referer := r.Header.Get("HX-Current-URL")
	if referer == "" {
		referer = r.Header.Get("Referer")
	}
	if referer == "" {
		referer = "/"
	}
	w.Header().Set("HX-Redirect", referer)
	w.WriteHeader(http.StatusOK)
}

func (h *PageSettingsHandlers) ClearConnectionAddress(w http.ResponseWriter, r *http.Request) {
	if err := h.settingsSvc.ClearConnectionAddress(); err != nil {
		h.log.Error("clearing connection address", "error", err)
		http.Error(w, "Failed to clear connection address", http.StatusInternalServerError)
		return
	}

	referer := r.Header.Get("HX-Current-URL")
	if referer == "" {
		referer = r.Header.Get("Referer")
	}
	if referer == "" {
		referer = "/"
	}
	w.Header().Set("HX-Redirect", referer)
	w.WriteHeader(http.StatusOK)
}
