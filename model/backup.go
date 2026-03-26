package model

import "time"

const (
	BackupStatusInProgress = "in_progress"
	BackupStatusCompleted  = "completed"
	BackupStatusFailed     = "failed"
)

type Backup struct {
	ID           string    `json:"id"`
	GameserverID string    `json:"gameserver_id"`
	Name         string    `json:"name"`
	SizeBytes    int64     `json:"size_bytes"`
	Status       string    `json:"status"`
	ErrorReason  string    `json:"error_reason,omitempty"`
	CreatedAt    time.Time `json:"created_at"`
}

type BackupFilter struct {
	GameserverID string
	Pagination
}
