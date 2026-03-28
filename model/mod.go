package model

import (
	"encoding/json"
	"time"
)

type InstalledMod struct {
	ID            string          `json:"id"`
	GameserverID  string          `json:"gameserver_id"`
	Source        string          `json:"source"`
	SourceID      string          `json:"source_id"`
	Category      string          `json:"category"`
	Name          string          `json:"name"`
	Version       string          `json:"version"`
	VersionID     string          `json:"version_id"`
	FilePath    string `json:"file_path"`
	FileName    string `json:"file_name"`
	DownloadURL string `json:"download_url"`
	FileHash    string `json:"file_hash"`
	Delivery    string `json:"delivery"` // "file", "manifest", "pack"
	AutoInstalled bool            `json:"auto_installed"`
	DependsOn     *string         `json:"depends_on,omitempty"`
	PackID        *string         `json:"pack_id,omitempty"`
	Metadata      json.RawMessage `json:"metadata"`
	InstalledAt   time.Time       `json:"installed_at"`
}

type PackExclusion struct {
	ID         string    `json:"id"`
	PackModID  string    `json:"pack_mod_id"`
	SourceID   string    `json:"source_id"`
	ExcludedAt time.Time `json:"excluded_at"`
}
