package model

import (
	"encoding/json"
	"time"
)

type Schedule struct {
	ID           string          `json:"id"`
	GameserverID string          `json:"gameserver_id"`
	Name         string          `json:"name"`
	Type         string          `json:"type"`
	CronExpr     string          `json:"cron_expr"`
	Payload      json.RawMessage `json:"payload"`
	Enabled      bool            `json:"enabled"`
	OneShot      bool            `json:"one_shot"`
	LastRun      *time.Time      `json:"last_run"`
	NextRun      *time.Time      `json:"next_run"`
	CreatedAt    time.Time       `json:"created_at"`
}
