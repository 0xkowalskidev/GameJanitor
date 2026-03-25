package handlers

import (
	"encoding/json"
	"log/slog"
	"net/http"

	"github.com/warsmite/gamejanitor/models"
	"github.com/warsmite/gamejanitor/service"
)

type StatusHandlers struct {
	gameserverSvc *service.GameserverService
	querySvc      *service.QueryService
	workerSvc     *service.WorkerNodeService
	log           *slog.Logger
}

func NewStatusHandlers(gameserverSvc *service.GameserverService, querySvc *service.QueryService, workerSvc *service.WorkerNodeService, log *slog.Logger) *StatusHandlers {
	return &StatusHandlers{gameserverSvc: gameserverSvc, querySvc: querySvc, workerSvc: workerSvc, log: log}
}

type clusterStatus struct {
	Workers            int     `json:"workers"`
	WorkersCordoned    int     `json:"workers_cordoned"`
	TotalMemoryMB      int64   `json:"total_memory_mb"`
	AllocatedMemoryMB  int     `json:"allocated_memory_mb"`
	TotalCPU           float64 `json:"total_cpu"`
	AllocatedCPU       float64 `json:"allocated_cpu"`
	TotalStorageMB     int64   `json:"total_storage_mb"`
	AllocatedStorageMB int     `json:"allocated_storage_mb"`
}

type gameserverStatus struct {
	Total      int `json:"total"`
	Running    int `json:"running"`
	Stopped    int `json:"stopped"`
	Installing int `json:"installing"`
	Error      int `json:"error"`
}

func (h *StatusHandlers) Get(w http.ResponseWriter, r *http.Request) {
	filter := models.GameserverFilter{}
	if token := service.TokenFromContext(r.Context()); token != nil {
		var gsIDs []string
		if err := json.Unmarshal(token.GameserverIDs, &gsIDs); err == nil && len(gsIDs) > 0 {
			filter.IDs = gsIDs
		}
	}

	gameservers, err := h.gameserverSvc.ListGameservers(filter)
	if err != nil {
		h.log.Error("listing gameservers for status", "error", err)
		respondError(w, http.StatusInternalServerError, err.Error())
		return
	}

	gs := gameserverStatus{Total: len(gameservers)}
	for _, g := range gameservers {
		switch g.Status {
		case service.StatusRunning, service.StatusStarted:
			gs.Running++
		case service.StatusStopped:
			gs.Stopped++
		case service.StatusInstalling:
			gs.Installing++
		case service.StatusError:
			gs.Error++
		}
	}

	cluster := h.buildClusterStatus()

	respondOK(w, map[string]any{
		"cluster":     cluster,
		"gameservers": gs,
	})
}

func (h *StatusHandlers) buildClusterStatus() clusterStatus {
	var cs clusterStatus

	workers, err := h.workerSvc.List()
	if err != nil {
		h.log.Error("listing workers for cluster status", "error", err)
		return cs
	}

	cs.Workers = len(workers)
	for _, w := range workers {
		cs.TotalMemoryMB += w.MemoryTotalMB
		cs.AllocatedMemoryMB += w.AllocatedMemoryMB
		cs.TotalCPU += float64(w.CPUCores)
		cs.AllocatedCPU += w.AllocatedCPU
		cs.TotalStorageMB += w.DiskTotalMB
		cs.AllocatedStorageMB += w.AllocatedStorageMB
		if w.Cordoned {
			cs.WorkersCordoned++
		}
	}

	return cs
}
