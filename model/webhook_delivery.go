package model

import "time"

const (
	WebhookStatePending   = "pending"
	WebhookStateDelivered = "delivered"
	WebhookStateFailed    = "failed"
)

type WebhookDelivery struct {
	ID                string
	WebhookEndpointID string
	EventType         string
	Payload           string // pre-serialized JSON
	State             string
	Attempts          int
	LastAttemptAt     *time.Time
	NextAttemptAt     time.Time
	LastError         string
	CreatedAt         time.Time
}
