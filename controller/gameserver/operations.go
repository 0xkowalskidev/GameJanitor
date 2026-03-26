package gameserver

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"time"

	"github.com/google/uuid"
	"github.com/warsmite/gamejanitor/model"
)

type opContextKey struct{}

// WithOperationID attaches an operation ID to the context.
// Inner operations (e.g. Stop/Start within Restart) check this to avoid
// creating nested operations.
func WithOperationID(ctx context.Context, opID string) context.Context {
	return context.WithValue(ctx, opContextKey{}, opID)
}

// OperationIDFromContext returns the operation ID from context, if any.
func OperationIDFromContext(ctx context.Context) string {
	if v, ok := ctx.Value(opContextKey{}).(string); ok {
		return v
	}
	return ""
}

// OperationStore abstracts operation persistence.
type OperationStore interface {
	CreateOperation(op *model.Operation) error
	CompleteOperation(id string) error
	FailOperation(id string, errMsg string) error
	HasRunningOperation(gameserverID string) (bool, error)
}

// OperationTracker manages the lifecycle of operations.
// Shared between GameserverService and BackupService.
type OperationTracker struct {
	store OperationStore
	log   *slog.Logger
}

func NewOperationTracker(store OperationStore, log *slog.Logger) *OperationTracker {
	return &OperationTracker{store: store, log: log}
}

// Start records a new running operation. Returns the operation ID.
// Returns an error if the gameserver already has a running operation
// (unless the new operation is a stop — you should always be able to stop).
func (t *OperationTracker) Start(gameserverID, workerID, opType string, metadata any) (string, error) {
	if opType != model.OpStop {
		busy, err := t.store.HasRunningOperation(gameserverID)
		if err != nil {
			return "", err
		}
		if busy {
			return "", fmt.Errorf("gameserver %s already has an operation in progress", gameserverID)
		}
	}

	metaJSON, _ := json.Marshal(metadata)
	op := &model.Operation{
		ID:           uuid.New().String(),
		GameserverID: gameserverID,
		WorkerID:     workerID,
		Type:         opType,
		Status:       model.OperationStatusRunning,
		Metadata:     metaJSON,
		StartedAt:    time.Now(),
	}

	if err := t.store.CreateOperation(op); err != nil {
		t.log.Error("failed to create operation record", "type", opType, "gameserver_id", gameserverID, "error", err)
		return "", err
	}

	t.log.Info("operation started", "op_id", op.ID, "type", opType, "gameserver_id", gameserverID, "worker_id", workerID)
	return op.ID, nil
}

// Complete marks an operation as completed.
func (t *OperationTracker) Complete(opID string) {
	if opID == "" {
		return
	}
	if err := t.store.CompleteOperation(opID); err != nil {
		t.log.Error("failed to complete operation", "op_id", opID, "error", err)
	}
}

// Fail marks an operation as failed with an error message.
func (t *OperationTracker) Fail(opID string, reason error) {
	if opID == "" {
		return
	}
	errMsg := ""
	if reason != nil {
		errMsg = reason.Error()
	}
	if err := t.store.FailOperation(opID, errMsg); err != nil {
		t.log.Error("failed to fail operation", "op_id", opID, "error", err)
	}
}
