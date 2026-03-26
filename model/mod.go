package model

import (
	"encoding/json"
	"time"
)

type InstalledMod struct {
	ID           string          `json:"id"`
	GameserverID string          `json:"gameserver_id"`
	Source       string          `json:"source"`
	SourceID     string          `json:"source_id"`
	Name         string          `json:"name"`
	Version      string          `json:"version"`
	VersionID    string          `json:"version_id"`
	FilePath     string          `json:"file_path"`
	FileName     string          `json:"file_name"`
	Metadata     json.RawMessage `json:"metadata"`
	InstalledAt  time.Time       `json:"installed_at"`
}
