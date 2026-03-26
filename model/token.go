package model

import (
	"encoding/json"
	"time"
)

type Token struct {
	ID            string          `json:"id"`
	Name          string          `json:"name"`
	HashedToken   string          `json:"-"`
	TokenPrefix   string          `json:"-"`
	Scope         string          `json:"scope"`
	GameserverIDs json.RawMessage `json:"gameserver_ids"`
	Permissions   json.RawMessage `json:"permissions"`
	CreatedAt     time.Time       `json:"created_at"`
	LastUsedAt    *time.Time      `json:"last_used_at,omitempty"`
	ExpiresAt     *time.Time      `json:"expires_at,omitempty"`
}
