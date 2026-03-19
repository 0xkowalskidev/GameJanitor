package worker

import (
	"database/sql"
	"fmt"
	"log/slog"
	"math"

	"github.com/0xkowalskidev/gamejanitor/internal/models"
)

// Dispatcher routes operations to the correct Worker for a given gameserver.
// In standalone mode, all operations go to a single LocalWorker.
// In multi-node mode, routes based on gameserver-to-node assignment.
type Dispatcher struct {
	local    Worker    // nil if controller-only (no local Docker)
	registry *Registry // nil in standalone mode
	db       *sql.DB   // nil in standalone mode (no node lookups needed)
	log      *slog.Logger
}

// NewLocalDispatcher creates a standalone dispatcher that routes everything to a local worker.
func NewLocalDispatcher(w Worker) *Dispatcher {
	return &Dispatcher{local: w}
}

// NewMultiNodeDispatcher creates a dispatcher that routes to local or remote workers
// based on the gameserver's node_id in the database.
func NewMultiNodeDispatcher(local Worker, registry *Registry, db *sql.DB, log *slog.Logger) *Dispatcher {
	return &Dispatcher{
		local:    local,
		registry: registry,
		db:       db,
		log:      log,
	}
}

// WorkerFor returns the Worker responsible for an existing gameserver.
// In standalone mode, always returns the local worker.
// In multi-node mode, looks up the gameserver's node_id and routes accordingly.
func (d *Dispatcher) WorkerFor(gameserverID string) Worker {
	if d.registry == nil {
		return d.local
	}

	nodeID, err := d.lookupNodeID(gameserverID)
	if err != nil {
		d.log.Error("looking up node for gameserver, falling back to local", "gameserver_id", gameserverID, "error", err)
		return d.local
	}

	// Empty node_id means local
	if nodeID == "" {
		if d.local != nil {
			return d.local
		}
		d.log.Error("gameserver assigned to local node but no local worker available", "gameserver_id", gameserverID)
		return nil
	}

	w, ok := d.registry.Get(nodeID)
	if !ok {
		d.log.Error("worker not found in registry", "node_id", nodeID, "gameserver_id", gameserverID)
		return nil
	}
	return w
}

// SelectWorkerForPlacement picks the best worker for new gameserver creation.
// Ranks by least allocated memory (sum of memory_limit_mb for assigned gameservers),
// not live free memory, to avoid overcommit when stopped servers are started.
func (d *Dispatcher) SelectWorkerForPlacement() (Worker, string) {
	if d.registry == nil {
		return d.local, ""
	}

	workers := d.registry.ListWorkers()
	if len(workers) == 0 {
		if d.local == nil {
			d.log.Error("no workers available for gameserver placement")
			return nil, ""
		}
		return d.local, ""
	}

	var bestNodeID string
	bestHeadroom := math.MinInt64

	for _, info := range workers {
		allocated, err := models.AllocatedMemoryByNode(d.db, info.ID)
		if err != nil {
			d.log.Warn("failed to query allocated memory for worker", "worker_id", info.ID, "error", err)
			continue
		}

		// If MaxMemoryMB is set, headroom = limit - allocated. Otherwise use negative allocated
		// so the worker with least allocation wins.
		node, _ := models.GetWorkerNode(d.db, info.ID)
		var headroom int
		if node != nil && node.MaxMemoryMB != nil {
			headroom = *node.MaxMemoryMB - allocated
		} else {
			headroom = -allocated
		}

		if headroom > bestHeadroom {
			bestHeadroom = headroom
			bestNodeID = info.ID
		}
	}

	if bestNodeID == "" {
		d.log.Warn("no suitable worker found, falling back to local")
		if d.local == nil {
			return nil, ""
		}
		return d.local, ""
	}

	w, ok := d.registry.Get(bestNodeID)
	if !ok {
		d.log.Warn("best worker disappeared from registry, falling back to local", "worker_id", bestNodeID)
		if d.local == nil {
			return nil, ""
		}
		return d.local, ""
	}

	d.log.Debug("selected worker for placement", "worker_id", bestNodeID, "headroom_mb", bestHeadroom)
	return w, bestNodeID
}

// SelectWorkerByNodeID returns the Worker for a specific node ID.
// Used when the user explicitly chooses a node for placement.
func (d *Dispatcher) SelectWorkerByNodeID(nodeID string) (Worker, error) {
	if nodeID == "" {
		if d.local != nil {
			return d.local, nil
		}
		return nil, fmt.Errorf("no local worker available")
	}

	if d.registry == nil {
		return nil, fmt.Errorf("multi-node not enabled")
	}

	w, ok := d.registry.Get(nodeID)
	if !ok {
		return nil, fmt.Errorf("worker %s is not connected", nodeID)
	}
	return w, nil
}

// ListWorkers returns info for all registered workers. Returns nil in standalone mode.
func (d *Dispatcher) ListWorkers() []WorkerInfo {
	if d.registry == nil {
		return nil
	}
	return d.registry.ListWorkers()
}

func (d *Dispatcher) lookupNodeID(gameserverID string) (string, error) {
	if d.db == nil {
		return "", nil
	}
	var nodeID sql.NullString
	err := d.db.QueryRow("SELECT node_id FROM gameservers WHERE id = ?", gameserverID).Scan(&nodeID)
	if err != nil {
		return "", fmt.Errorf("querying node_id for gameserver %s: %w", gameserverID, err)
	}
	return nodeID.String, nil
}
