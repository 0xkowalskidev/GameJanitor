package handlers

import (
	"database/sql"
	"log/slog"
	"net/http"
	"strconv"

	"github.com/warsmite/gamejanitor/internal/models"
)

type AuditHandlers struct {
	db  *sql.DB
	log *slog.Logger
}

func NewAuditHandlers(db *sql.DB, log *slog.Logger) *AuditHandlers {
	return &AuditHandlers{db: db, log: log}
}

func (h *AuditHandlers) List(w http.ResponseWriter, r *http.Request) {
	filter := models.AuditLogFilter{}
	if v := r.URL.Query().Get("action"); v != "" {
		filter.Action = &v
	}
	if v := r.URL.Query().Get("resource_type"); v != "" {
		filter.ResourceType = &v
	}
	if v := r.URL.Query().Get("resource_id"); v != "" {
		filter.ResourceID = &v
	}
	if v := r.URL.Query().Get("token_id"); v != "" {
		filter.TokenID = &v
	}
	if v := r.URL.Query().Get("limit"); v != "" {
		if n, err := strconv.Atoi(v); err == nil {
			filter.Limit = n
		}
	}
	if v := r.URL.Query().Get("offset"); v != "" {
		if n, err := strconv.Atoi(v); err == nil {
			filter.Offset = n
		}
	}

	entries, err := models.ListAuditLogs(h.db, filter)
	if err != nil {
		h.log.Error("listing audit logs", "error", err)
		respondError(w, http.StatusInternalServerError, err.Error())
		return
	}
	if entries == nil {
		entries = []models.AuditLog{}
	}
	respondOK(w, entries)
}
