package handlers

import (
	"bytes"
	"crypto/hmac"
	"crypto/sha256"
	"database/sql"
	"encoding/hex"
	"encoding/json"
	"io"
	"log/slog"
	"net/http"
	"path"
	"strings"
	"time"

	"github.com/0xkowalskidev/gamejanitor/internal/models"
	"github.com/0xkowalskidev/gamejanitor/internal/service"
	"github.com/go-chi/chi/v5"
)

type WebhookHandlers struct {
	db  *sql.DB
	log *slog.Logger
}

func NewWebhookHandlers(db *sql.DB, log *slog.Logger) *WebhookHandlers {
	return &WebhookHandlers{db: db, log: log}
}

type webhookEndpointResponse struct {
	ID          string    `json:"id"`
	Description string    `json:"description"`
	URL         string    `json:"url"`
	SecretSet   bool      `json:"secret_set"`
	Events      []string  `json:"events"`
	Enabled     bool      `json:"enabled"`
	CreatedAt   time.Time `json:"created_at"`
	UpdatedAt   time.Time `json:"updated_at"`
}

func toWebhookResponse(e *models.WebhookEndpoint) webhookEndpointResponse {
	var events []string
	if err := json.Unmarshal([]byte(e.Events), &events); err != nil {
		events = []string{}
	}
	return webhookEndpointResponse{
		ID:          e.ID,
		Description: e.Description,
		URL:         e.URL,
		SecretSet:   e.Secret != "",
		Events:      events,
		Enabled:     e.Enabled,
		CreatedAt:   e.CreatedAt,
		UpdatedAt:   e.UpdatedAt,
	}
}

func (h *WebhookHandlers) List(w http.ResponseWriter, r *http.Request) {
	endpoints, err := models.ListWebhookEndpoints(h.db)
	if err != nil {
		respondError(w, http.StatusInternalServerError, "failed to list webhook endpoints")
		return
	}

	result := make([]webhookEndpointResponse, 0, len(endpoints))
	for _, e := range endpoints {
		result = append(result, toWebhookResponse(&e))
	}
	respondOK(w, result)
}

func (h *WebhookHandlers) Get(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "webhookId")
	ep, err := models.GetWebhookEndpoint(h.db, id)
	if err != nil {
		respondError(w, http.StatusInternalServerError, "failed to get webhook endpoint")
		return
	}
	if ep == nil {
		respondError(w, http.StatusNotFound, "webhook endpoint not found")
		return
	}
	respondOK(w, toWebhookResponse(ep))
}

type createWebhookRequest struct {
	Description string   `json:"description"`
	URL         string   `json:"url"`
	Secret      string   `json:"secret"`
	Events      []string `json:"events"`
	Enabled     *bool    `json:"enabled"`
}

func (h *WebhookHandlers) Create(w http.ResponseWriter, r *http.Request) {
	var req createWebhookRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		respondError(w, http.StatusBadRequest, "invalid JSON: "+err.Error())
		return
	}

	if req.URL == "" {
		respondError(w, http.StatusBadRequest, "url is required")
		return
	}
	if !strings.HasPrefix(req.URL, "http://") && !strings.HasPrefix(req.URL, "https://") {
		respondError(w, http.StatusBadRequest, "url must start with http:// or https://")
		return
	}

	events := req.Events
	if len(events) == 0 {
		events = []string{"*"}
	}
	if err := validateEventFilter(events); err != nil {
		respondError(w, http.StatusBadRequest, err.Error())
		return
	}

	eventsJSON, _ := json.Marshal(events)

	enabled := true
	if req.Enabled != nil {
		enabled = *req.Enabled
	}

	ep := &models.WebhookEndpoint{
		Description: req.Description,
		URL:         req.URL,
		Secret:      req.Secret,
		Events:      string(eventsJSON),
		Enabled:     enabled,
	}
	if err := models.CreateWebhookEndpoint(h.db, ep); err != nil {
		respondError(w, http.StatusInternalServerError, "failed to create webhook endpoint")
		return
	}

	h.log.Info("webhook endpoint created", "id", ep.ID, "url", ep.URL)
	respondCreated(w, toWebhookResponse(ep))
}

type updateWebhookRequest struct {
	Description *string  `json:"description"`
	URL         *string  `json:"url"`
	Secret      *string  `json:"secret"`
	Events      []string `json:"events"`
	Enabled     *bool    `json:"enabled"`
}

func (h *WebhookHandlers) Update(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "webhookId")
	ep, err := models.GetWebhookEndpoint(h.db, id)
	if err != nil {
		respondError(w, http.StatusInternalServerError, "failed to get webhook endpoint")
		return
	}
	if ep == nil {
		respondError(w, http.StatusNotFound, "webhook endpoint not found")
		return
	}

	var req updateWebhookRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		respondError(w, http.StatusBadRequest, "invalid JSON: "+err.Error())
		return
	}

	if req.Description != nil {
		ep.Description = *req.Description
	}
	if req.URL != nil {
		if !strings.HasPrefix(*req.URL, "http://") && !strings.HasPrefix(*req.URL, "https://") {
			respondError(w, http.StatusBadRequest, "url must start with http:// or https://")
			return
		}
		ep.URL = *req.URL
	}
	if req.Secret != nil {
		ep.Secret = *req.Secret
	}
	if req.Events != nil {
		if err := validateEventFilter(req.Events); err != nil {
			respondError(w, http.StatusBadRequest, err.Error())
			return
		}
		eventsJSON, _ := json.Marshal(req.Events)
		ep.Events = string(eventsJSON)
	}
	if req.Enabled != nil {
		ep.Enabled = *req.Enabled
	}

	if err := models.UpdateWebhookEndpoint(h.db, ep); err != nil {
		respondError(w, http.StatusInternalServerError, "failed to update webhook endpoint")
		return
	}

	h.log.Info("webhook endpoint updated", "id", ep.ID)
	respondOK(w, toWebhookResponse(ep))
}

func (h *WebhookHandlers) Delete(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "webhookId")
	if err := models.DeleteWebhookEndpoint(h.db, id); err != nil {
		if err == sql.ErrNoRows {
			respondError(w, http.StatusNotFound, "webhook endpoint not found")
			return
		}
		respondError(w, http.StatusInternalServerError, "failed to delete webhook endpoint")
		return
	}

	h.log.Info("webhook endpoint deleted", "id", id)
	respondNoContent(w)
}

func (h *WebhookHandlers) Test(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "webhookId")
	ep, err := models.GetWebhookEndpoint(h.db, id)
	if err != nil {
		respondError(w, http.StatusInternalServerError, "failed to get webhook endpoint")
		return
	}
	if ep == nil {
		respondError(w, http.StatusNotFound, "webhook endpoint not found")
		return
	}

	payload := service.WebhookPayload{
		ID:        "test",
		Timestamp: time.Now().UTC(),
		EventType: "webhook.test",
		Data:      map[string]string{},
	}

	body, err := json.Marshal(payload)
	if err != nil {
		respondError(w, http.StatusInternalServerError, "failed to marshal test payload")
		return
	}

	statusCode, deliverErr := deliverWebhook(ep.URL, body, ep.Secret)
	if deliverErr != nil {
		respondError(w, http.StatusBadGateway, deliverErr.Error())
		return
	}

	respondOK(w, map[string]any{
		"response_status": statusCode,
		"success":         statusCode >= 200 && statusCode < 300,
	})
}

func deliverWebhook(url string, body []byte, secret string) (int, error) {
	client := &http.Client{Timeout: 10 * time.Second}

	req, err := http.NewRequest(http.MethodPost, url, bytes.NewReader(body))
	if err != nil {
		return 0, err
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("User-Agent", "Gamejanitor-Webhook/1.0")

	if secret != "" {
		mac := hmac.New(sha256.New, []byte(secret))
		mac.Write(body)
		sig := hex.EncodeToString(mac.Sum(nil))
		req.Header.Set("X-Webhook-Signature", "sha256="+sig)
	}

	resp, err := client.Do(req)
	if err != nil {
		return 0, err
	}
	defer resp.Body.Close()
	io.Copy(io.Discard, resp.Body)

	return resp.StatusCode, nil
}

// Known event types for validation
var knownEventTypes = []string{
	"status_changed",
	"gameserver.created", "gameserver.updated", "gameserver.deleted",
	"backup.created", "backup.deleted", "backup.restore_completed", "backup.restore_failed",
	"worker.connected", "worker.disconnected",
	"schedule.task_completed", "schedule.task_failed",
}

func validateEventFilter(events []string) error {
	if len(events) == 0 {
		return service.ErrBadRequestf("events must not be empty")
	}
	for _, e := range events {
		if e == "*" {
			continue
		}
		// Check if it matches at least one known event type (literal or glob)
		matched := false
		for _, known := range knownEventTypes {
			if e == known {
				matched = true
				break
			}
			if m, _ := path.Match(e, known); m {
				matched = true
				break
			}
		}
		if !matched {
			return service.ErrBadRequestf("event filter %q does not match any known event types", e)
		}
	}
	return nil
}
