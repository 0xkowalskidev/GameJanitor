package service

import (
	"database/sql"
	"fmt"
	"log/slog"
	"strconv"

	"github.com/warsmite/gamejanitor/internal/models"
)

type SettingType int

const (
	SettingTypeBool SettingType = iota
	SettingTypeInt
	SettingTypeString
)

type SettingDef struct {
	Key     string
	Type    SettingType
	Default any
}

// Setting key constants
const (
	SettingConnectionAddress  = "connection_address"
	SettingPortRangeStart     = "port_range_start"
	SettingPortRangeEnd       = "port_range_end"
	SettingPortMode           = "port_mode"
	SettingMaxBackups         = "max_backups"
	SettingAuthEnabled        = "auth_enabled"
	SettingLocalhostBypass    = "localhost_bypass"
	SettingRateLimitEnabled   = "rate_limit_enabled"
	SettingRateLimitPerIP     = "rate_limit_per_ip"
	SettingRateLimitPerToken  = "rate_limit_per_token"
	SettingRateLimitLogin     = "rate_limit_login"
	SettingTrustProxyHeaders  = "trust_proxy_headers"
	SettingEventRetention     = "event_retention_days"
	SettingRequireMemoryLimit = "require_memory_limit"
	SettingRequireCPULimit    = "require_cpu_limit"
	SettingRequireStorageLimit = "require_storage_limit"
)

// Registry of all runtime settings with their types and defaults.
var SettingDefs = []SettingDef{
	{SettingConnectionAddress, SettingTypeString, ""},
	{SettingPortRangeStart, SettingTypeInt, 27000},
	{SettingPortRangeEnd, SettingTypeInt, 28999},
	{SettingPortMode, SettingTypeString, "auto"},
	{SettingMaxBackups, SettingTypeInt, 10},
	{SettingAuthEnabled, SettingTypeBool, false},
	{SettingLocalhostBypass, SettingTypeBool, true},
	{SettingRateLimitEnabled, SettingTypeBool, false},
	{SettingRateLimitPerIP, SettingTypeInt, 20},
	{SettingRateLimitPerToken, SettingTypeInt, 10},
	{SettingRateLimitLogin, SettingTypeInt, 10},
	{SettingTrustProxyHeaders, SettingTypeBool, false},
	{SettingEventRetention, SettingTypeInt, 30},
	{SettingRequireMemoryLimit, SettingTypeBool, false},
	{SettingRequireCPULimit, SettingTypeBool, false},
	{SettingRequireStorageLimit, SettingTypeBool, false},
}

type SettingsService struct {
	db   *sql.DB
	log  *slog.Logger
	defs map[string]SettingDef
}

func NewSettingsService(db *sql.DB, log *slog.Logger) *SettingsService {
	defs := make(map[string]SettingDef, len(SettingDefs))
	for _, d := range SettingDefs {
		defs[d.Key] = d
	}
	return &SettingsService{db: db, log: log, defs: defs}
}

// ApplyConfig writes config-specified settings to DB on startup.
// Only keys present in the map are written — unspecified settings are left alone.
func (s *SettingsService) ApplyConfig(settings map[string]any) {
	if len(settings) == 0 {
		return
	}

	applied := 0
	for key, val := range settings {
		def, ok := s.defs[key]
		if !ok {
			s.log.Warn("ignoring unknown setting from config", "key", key)
			continue
		}

		var strVal string
		switch def.Type {
		case SettingTypeBool:
			b, ok := toBool(val)
			if !ok {
				s.log.Warn("invalid bool value for setting", "key", key, "value", val)
				continue
			}
			strVal = strconv.FormatBool(b)
		case SettingTypeInt:
			n, ok := toInt(val)
			if !ok {
				s.log.Warn("invalid int value for setting", "key", key, "value", val)
				continue
			}
			strVal = strconv.Itoa(n)
		case SettingTypeString:
			strVal = fmt.Sprintf("%v", val)
		}

		if err := models.SetSetting(s.db, key, strVal); err != nil {
			s.log.Error("failed to apply config setting to DB", "key", key, "error", err)
			continue
		}
		applied++
	}

	if applied > 0 {
		s.log.Info("applied config settings to DB", "count", applied)
	}
}

// GetBool returns a boolean setting value. Falls back to the registered default.
func (s *SettingsService) GetBool(key string) bool {
	v, err := models.GetSetting(s.db, key)
	if err != nil || v == "" {
		if def, ok := s.defs[key]; ok {
			if b, ok := def.Default.(bool); ok {
				return b
			}
		}
		return false
	}
	return v == "true"
}

// GetInt returns an integer setting value. Falls back to the registered default.
func (s *SettingsService) GetInt(key string) int {
	v, err := models.GetSetting(s.db, key)
	if err != nil || v == "" {
		if def, ok := s.defs[key]; ok {
			if n, ok := toInt(def.Default); ok {
				return n
			}
		}
		return 0
	}
	n, err := strconv.Atoi(v)
	if err != nil {
		if def, ok := s.defs[key]; ok {
			if n, ok := toInt(def.Default); ok {
				return n
			}
		}
		return 0
	}
	return n
}

// GetString returns a string setting value. Falls back to the registered default.
func (s *SettingsService) GetString(key string) string {
	v, err := models.GetSetting(s.db, key)
	if err != nil || v == "" {
		if def, ok := s.defs[key]; ok {
			if str, ok := def.Default.(string); ok {
				return str
			}
		}
		return ""
	}
	return v
}

// Set writes a setting value to DB after validating the key exists.
func (s *SettingsService) Set(key string, value any) error {
	def, ok := s.defs[key]
	if !ok {
		return fmt.Errorf("unknown setting: %s", key)
	}

	var strVal string
	switch def.Type {
	case SettingTypeBool:
		b, ok := toBool(value)
		if !ok {
			return fmt.Errorf("invalid bool value for %s", key)
		}
		strVal = strconv.FormatBool(b)
	case SettingTypeInt:
		n, ok := toInt(value)
		if !ok {
			return fmt.Errorf("invalid int value for %s", key)
		}
		strVal = strconv.Itoa(n)
	case SettingTypeString:
		strVal = fmt.Sprintf("%v", value)
	}

	return models.SetSetting(s.db, key, strVal)
}

// Clear removes a setting from DB, reverting it to its default.
func (s *SettingsService) Clear(key string) error {
	return models.DeleteSetting(s.db, key)
}

// All returns all settings with their current values.
func (s *SettingsService) All() map[string]any {
	result := make(map[string]any, len(s.defs))
	for _, def := range SettingDefs {
		switch def.Type {
		case SettingTypeBool:
			result[def.Key] = s.GetBool(def.Key)
		case SettingTypeInt:
			result[def.Key] = s.GetInt(def.Key)
		case SettingTypeString:
			result[def.Key] = s.GetString(def.Key)
		}
	}
	return result
}

// Def returns the setting definition for a key, if it exists.
func (s *SettingsService) Def(key string) (SettingDef, bool) {
	d, ok := s.defs[key]
	return d, ok
}

// ResolveConnectionIP returns the connection IP for a gameserver on the given node.
// Priority: global override > worker external IP > worker LAN IP > empty (caller falls back to 127.0.0.1).
func (s *SettingsService) ResolveConnectionIP(nodeID *string) (ip string, configured bool) {
	if globalIP := s.GetString(SettingConnectionAddress); globalIP != "" {
		return globalIP, true
	}
	if nodeID != nil && *nodeID != "" {
		node, err := models.GetWorkerNode(s.db, *nodeID)
		if err == nil && node != nil {
			if node.ExternalIP != "" {
				return node.ExternalIP, true
			}
			if node.LanIP != "" {
				return node.LanIP, true
			}
		}
	}
	return "", false
}

// toBool converts various YAML-parsed types to bool.
func toBool(v any) (bool, bool) {
	switch val := v.(type) {
	case bool:
		return val, true
	case string:
		return val == "true" || val == "1", true
	default:
		return false, false
	}
}

// toInt converts various YAML-parsed types to int.
func toInt(v any) (int, bool) {
	switch val := v.(type) {
	case int:
		return val, true
	case int64:
		return int(val), true
	case float64:
		return int(val), true
	case string:
		n, err := strconv.Atoi(val)
		return n, err == nil
	default:
		return 0, false
	}
}
