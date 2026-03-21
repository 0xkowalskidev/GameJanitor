package service

import (
	"database/sql"
	"fmt"
	"log/slog"
	"time"

	"github.com/0xkowalskidev/gamejanitor/internal/models"
)

// Gameserver status constants.
// Lifecycle: Stopped → Pulling → Starting → Started → Running → Stopping → Stopped
// Operations: Updating, Reinstalling, Migrating, Restoring (set before stop, cleared when Start() takes over)
// Any state can transition to Error on unexpected failures.
const (
	StatusStopped      = "stopped"
	StatusPulling      = "pulling"
	StatusStarting     = "starting"
	StatusStarted      = "started"
	StatusRunning      = "running"
	StatusStopping     = "stopping"
	StatusError        = "error"
	StatusUpdating     = "updating"
	StatusReinstalling = "reinstalling"
	StatusMigrating    = "migrating"
	StatusRestoring    = "restoring"
)

// setGameserverStatus updates a gameserver's status in the DB and logs the transition.
// When transitioning to error, errorReason is stored; otherwise it's cleared.
func setGameserverStatus(db *sql.DB, log *slog.Logger, broadcaster *EventBroadcaster, id string, newStatus string, errorReason string) error {
	gs, err := models.GetGameserver(db, id)
	if err != nil {
		return err
	}
	if gs == nil {
		return ErrNotFoundf("gameserver %s not found", id)
	}

	oldStatus := gs.Status
	gs.Status = newStatus
	if newStatus == StatusError {
		gs.ErrorReason = errorReason
	} else {
		gs.ErrorReason = ""
	}
	if err := models.UpdateGameserver(db, gs); err != nil {
		return fmt.Errorf("updating gameserver %s status from %s to %s: %w", id, oldStatus, newStatus, err)
	}

	log.Info("gameserver status changed", "id", id, "from", oldStatus, "to", newStatus)

	if broadcaster != nil {
		broadcaster.Publish(StatusEvent{
			GameserverID: id,
			OldStatus:    oldStatus,
			NewStatus:    newStatus,
			ErrorReason:  gs.ErrorReason,
			Timestamp:    time.Now(),
		})
	}

	return nil
}

func needsRecovery(status string) bool {
	return status != StatusStopped && status != StatusError
}

func isRunningStatus(status string) bool {
	return status == StatusStarted || status == StatusRunning
}

// isOperationStatus returns true for statuses that represent a multi-step
// operation in progress. Stop() preserves these so the UI shows what's
// actually happening instead of a generic "stopping" → "stopped" flicker.
func isOperationStatus(status string) bool {
	return status == StatusUpdating || status == StatusReinstalling || status == StatusMigrating || status == StatusRestoring
}
