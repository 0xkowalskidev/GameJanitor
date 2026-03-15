package worker

import (
	"fmt"
	"log/slog"
	"sync"
	"time"
)

// WorkerInfo tracks a connected worker's metadata and status.
type WorkerInfo struct {
	ID               string
	LanIP            string
	ExternalIP       string
	CPUCores         int64
	MemoryTotalMB    int64
	MemoryAvailableMB int64
	LastSeen         time.Time
}

// Registry tracks connected remote workers.
// Used by the controller in multi-node mode.
type Registry struct {
	workers map[string]*registeredWorker
	mu      sync.RWMutex
	log     *slog.Logger
}

type registeredWorker struct {
	worker *RemoteWorker
	info   WorkerInfo
}

func NewRegistry(log *slog.Logger) *Registry {
	return &Registry{
		workers: make(map[string]*registeredWorker),
		log:     log,
	}
}

func (r *Registry) Register(w *RemoteWorker, info WorkerInfo) {
	r.mu.Lock()
	defer r.mu.Unlock()

	info.LastSeen = time.Now()
	r.workers[w.nodeID] = &registeredWorker{worker: w, info: info}
	r.log.Info("worker registered", "worker_id", w.nodeID, "lan_ip", info.LanIP, "external_ip", info.ExternalIP)
}

func (r *Registry) Unregister(nodeID string) {
	r.mu.Lock()
	defer r.mu.Unlock()

	if rw, ok := r.workers[nodeID]; ok {
		rw.worker.Close()
		delete(r.workers, nodeID)
		r.log.Info("worker unregistered", "worker_id", nodeID)
	}
}

func (r *Registry) Get(nodeID string) (Worker, bool) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	rw, ok := r.workers[nodeID]
	if !ok {
		return nil, false
	}
	return rw.worker, true
}

func (r *Registry) GetInfo(nodeID string) (WorkerInfo, bool) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	rw, ok := r.workers[nodeID]
	if !ok {
		return WorkerInfo{}, false
	}
	return rw.info, true
}

func (r *Registry) UpdateHeartbeat(nodeID string, info WorkerInfo) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	rw, ok := r.workers[nodeID]
	if !ok {
		return fmt.Errorf("worker %s not registered", nodeID)
	}
	info.LastSeen = time.Now()
	rw.info = info
	return nil
}

// ListWorkers returns info for all registered workers.
func (r *Registry) ListWorkers() []WorkerInfo {
	r.mu.RLock()
	defer r.mu.RUnlock()

	infos := make([]WorkerInfo, 0, len(r.workers))
	for _, rw := range r.workers {
		infos = append(infos, rw.info)
	}
	return infos
}

// BestWorker returns the worker with the most available memory.
// Used for placement decisions when creating new gameservers.
func (r *Registry) BestWorker() (Worker, string, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	if len(r.workers) == 0 {
		return nil, "", fmt.Errorf("no workers registered")
	}

	var best *registeredWorker
	for _, rw := range r.workers {
		if best == nil || rw.info.MemoryAvailableMB > best.info.MemoryAvailableMB {
			best = rw
		}
	}
	return best.worker, best.info.ID, nil
}

func (r *Registry) Count() int {
	r.mu.RLock()
	defer r.mu.RUnlock()
	return len(r.workers)
}
