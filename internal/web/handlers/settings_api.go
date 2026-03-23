package handlers

import (
	"encoding/json"
	"fmt"
	"log/slog"
	"net/http"

	"github.com/warsmite/gamejanitor/internal/service"
)

type SettingsAPIHandlers struct {
	settingsSvc *service.SettingsService
	log         *slog.Logger
}

func NewSettingsAPIHandlers(settingsSvc *service.SettingsService, log *slog.Logger) *SettingsAPIHandlers {
	return &SettingsAPIHandlers{settingsSvc: settingsSvc, log: log}
}

func (h *SettingsAPIHandlers) Get(w http.ResponseWriter, r *http.Request) {
	respondOK(w, h.settingsSvc.All())
}

func (h *SettingsAPIHandlers) Update(w http.ResponseWriter, r *http.Request) {
	var req map[string]json.RawMessage
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		respondError(w, http.StatusBadRequest, "invalid JSON: "+err.Error())
		return
	}

	for key, raw := range req {
		def, ok := h.settingsSvc.Def(key)
		if !ok {
			respondError(w, http.StatusBadRequest, "unknown setting: "+key)
			return
		}

		var value any
		switch def.Type {
		case service.SettingTypeBool:
			var v bool
			if err := json.Unmarshal(raw, &v); err != nil {
				respondError(w, http.StatusBadRequest, fmt.Sprintf("invalid bool value for %s", key))
				return
			}
			value = v
		case service.SettingTypeInt:
			var v int
			if err := json.Unmarshal(raw, &v); err != nil {
				respondError(w, http.StatusBadRequest, fmt.Sprintf("invalid int value for %s", key))
				return
			}
			value = v
		case service.SettingTypeString:
			var v string
			if err := json.Unmarshal(raw, &v); err != nil {
				respondError(w, http.StatusBadRequest, fmt.Sprintf("invalid string value for %s", key))
				return
			}
			// Empty string clears the setting
			if v == "" {
				if err := h.settingsSvc.Clear(key); err != nil {
					respondError(w, http.StatusInternalServerError, "failed to clear setting")
					return
				}
				continue
			}
			value = v
		}

		if err := h.settingsSvc.Set(key, value); err != nil {
			respondError(w, http.StatusInternalServerError, "failed to update setting: "+err.Error())
			return
		}
	}

	h.log.Info("settings updated via API", "fields", len(req))

	// Return current state after update
	h.Get(w, r)
}
