package handlers

import (
	"database/sql"
	"log/slog"
	"net/http"

	"github.com/warsmite/gamejanitor/internal/models"
)

type PageAuditHandlers struct {
	db       *sql.DB
	renderer *Renderer
	log      *slog.Logger
}

func NewPageAuditHandlers(db *sql.DB, renderer *Renderer, log *slog.Logger) *PageAuditHandlers {
	return &PageAuditHandlers{db: db, renderer: renderer, log: log}
}

func (h *PageAuditHandlers) List(w http.ResponseWriter, r *http.Request) {
	entries, err := models.ListAuditLogs(h.db, models.AuditLogFilter{Limit: 200})
	if err != nil {
		h.log.Error("listing audit logs for page", "error", err)
		h.renderer.RenderError(w, r, http.StatusInternalServerError)
		return
	}
	if entries == nil {
		entries = []models.AuditLog{}
	}

	// Collect unique actions for filter dropdown
	seen := make(map[string]bool)
	var uniqueActions []string
	for _, e := range entries {
		if !seen[e.Action] {
			seen[e.Action] = true
			uniqueActions = append(uniqueActions, e.Action)
		}
	}

	h.renderer.Render(w, r, "audit/index", map[string]any{
		"Entries":       entries,
		"UniqueActions": uniqueActions,
	})
}
