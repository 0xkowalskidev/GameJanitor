package gamejanitor

import (
	"context"
	"net/url"
)

// OperationService handles operation-related API calls.
type OperationService struct {
	client *Client
}

// OperationListOptions configures filters for listing operations.
type OperationListOptions struct {
	GameserverID string
	Status       string // running, completed, failed, abandoned
	WorkerID     string
}

// List returns operations matching the given filters.
func (s *OperationService) List(ctx context.Context, opts *OperationListOptions) ([]Operation, error) {
	v := url.Values{}
	if opts != nil {
		if opts.GameserverID != "" {
			v.Set("gameserver_id", opts.GameserverID)
		}
		if opts.Status != "" {
			v.Set("status", opts.Status)
		}
		if opts.WorkerID != "" {
			v.Set("worker_id", opts.WorkerID)
		}
	}
	path := "/api/operations"
	if len(v) > 0 {
		path += "?" + v.Encode()
	}

	var ops []Operation
	if err := s.client.get(ctx, path, &ops); err != nil {
		return nil, err
	}
	return ops, nil
}

// ListByGameserver is a convenience method to list operations for a specific gameserver.
func (s *OperationService) ListByGameserver(ctx context.Context, gameserverID string) ([]Operation, error) {
	return s.List(ctx, &OperationListOptions{GameserverID: gameserverID})
}

// Running returns all currently running operations.
func (s *OperationService) Running(ctx context.Context) ([]Operation, error) {
	return s.List(ctx, &OperationListOptions{Status: "running"})
}
