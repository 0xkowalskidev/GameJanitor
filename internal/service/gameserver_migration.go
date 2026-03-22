package service

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"time"

	"github.com/warsmite/gamejanitor/internal/models"
)

// MigrateGameserver moves a gameserver from its current node to a different node.
// Requires both source and target workers to be online.
func (s *GameserverService) MigrateGameserver(ctx context.Context, gameserverID string, targetNodeID string) (err error) {
	gs, err := models.GetGameserver(s.db, gameserverID)
	if err != nil {
		return err
	}
	if gs == nil {
		return ErrNotFoundf("gameserver %s not found", gameserverID)
	}

	currentNodeID := ""
	if gs.NodeID != nil {
		currentNodeID = *gs.NodeID
	}
	if currentNodeID == targetNodeID {
		return fmt.Errorf("gameserver is already on node %s", targetNodeID)
	}

	// Validate target worker is connected
	targetWorker, err := s.dispatcher.SelectWorkerByNodeID(targetNodeID)
	if err != nil {
		return fmt.Errorf("target worker unavailable: %w", err)
	}

	// Validate target node tags
	var requiredTags []string
	if gs.NodeTags != "" && gs.NodeTags != "[]" {
		json.Unmarshal([]byte(gs.NodeTags), &requiredTags)
	}
	if len(requiredTags) > 0 {
		targetNode, err := models.GetWorkerNode(s.db, targetNodeID)
		if err != nil || targetNode == nil {
			return fmt.Errorf("target node %s not found", targetNodeID)
		}
		var nodeTags []string
		json.Unmarshal([]byte(targetNode.Tags), &nodeTags)
		tagSet := make(map[string]bool, len(nodeTags))
		for _, t := range nodeTags {
			tagSet[t] = true
		}
		for _, req := range requiredTags {
			if !tagSet[req] {
				return fmt.Errorf("target node %s missing required tag: %s", targetNodeID, req)
			}
		}
	}

	// Check target node limits
	if err := s.checkWorkerLimits(targetNodeID, gs.MemoryLimitMB, gs.CPULimit, ptrIntOr0(gs.StorageLimitMB)); err != nil {
		return err
	}

	// Get source worker (must be online to transfer data)
	sourceWorker := s.dispatcher.WorkerFor(gameserverID)
	if sourceWorker == nil {
		return fmt.Errorf("source worker is offline, cannot migrate (both workers must be online)")
	}

	s.log.Info("migrating gameserver", "id", gameserverID, "from_node", currentNodeID, "to_node", targetNodeID)

	defer func() {
		if err != nil {
			s.broadcaster.Publish(GameserverErrorEvent{GameserverID: gameserverID, Reason: operationFailedReason("Migration failed", err), Timestamp: time.Now()})
		}
	}()

	// Stop if running
	if gs.Status != StatusStopped {
		s.log.Info("stopping gameserver for migration", "id", gameserverID)
		if err := s.Stop(ctx, gameserverID); err != nil {
			return fmt.Errorf("stopping gameserver for migration: %w", err)
		}
	}

	// Tar volume from source — fully buffer before modifying target to avoid
	// issues if source and target share a Docker daemon (same-host migration)
	s.log.Info("transferring volume data", "id", gameserverID, "volume", gs.VolumeName)
	tarReader, err := sourceWorker.BackupVolume(ctx, gs.VolumeName)
	if err != nil {
		return fmt.Errorf("reading volume from source worker: %w", err)
	}
	var tarBuf bytes.Buffer
	if _, err := io.Copy(&tarBuf, tarReader); err != nil {
		tarReader.Close()
		return fmt.Errorf("buffering volume data: %w", err)
	}
	tarReader.Close()
	s.log.Info("volume data buffered", "id", gameserverID, "size_bytes", tarBuf.Len())

	// Create volume on target and restore
	if err := targetWorker.CreateVolume(ctx, gs.VolumeName); err != nil {
		return fmt.Errorf("creating volume on target worker: %w", err)
	}

	if err := targetWorker.RestoreVolume(ctx, gs.VolumeName, &tarBuf); err != nil {
		// Clean up the volume we just created
		if rmErr := targetWorker.RemoveVolume(ctx, gs.VolumeName); rmErr != nil {
			s.log.Error("failed to clean up target volume after failed restore", "volume", gs.VolumeName, "error", rmErr)
		}
		return fmt.Errorf("restoring volume on target worker: %w", err)
	}

	// Reallocate ports on target node's range
	game := s.gameStore.GetGame(gs.GameID)
	if game == nil {
		return ErrNotFoundf("game %s not found", gs.GameID)
	}
	newPorts, err := s.AllocatePorts(game, targetNodeID, "")
	if err != nil {
		return fmt.Errorf("allocating ports on target node: %w", err)
	}

	// Update DB: node_id and ports
	gs.NodeID = &targetNodeID
	gs.Ports = newPorts
	if err := models.UpdateGameserver(s.db, gs); err != nil {
		return fmt.Errorf("updating gameserver node assignment: %w", err)
	}

	// Clean up old volume on source worker
	if err := sourceWorker.RemoveVolume(ctx, gs.VolumeName); err != nil {
		s.log.Warn("failed to remove old volume from source worker", "volume", gs.VolumeName, "error", err)
	}

	s.broadcaster.Publish(ContainerStoppedEvent{GameserverID: gameserverID, Timestamp: time.Now()})
	s.log.Info("gameserver migrated", "id", gameserverID, "from_node", currentNodeID, "to_node", targetNodeID)
	return nil
}
