package model

import (
	"encoding/json"
	"time"
)

type Event struct {
	ID           string          `json:"id"`
	EventType    string          `json:"event_type"`
	GameserverID string          `json:"gameserver_id,omitempty"`
	Actor        json.RawMessage `json:"actor"`
	Data         json.RawMessage `json:"data"`
	CreatedAt    time.Time       `json:"created_at"`
}

type EventFilter struct {
	EventType            string   // glob pattern
	GameserverID         string   // single gameserver filter
	AllowedGameserverIDs []string // token scope — only return events for these gameservers
	Pagination
}
