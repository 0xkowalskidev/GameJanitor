package handlers

import (
	"log/slog"
	"net/http"
	"time"

	"github.com/0xkowalskidev/gamejanitor/internal/models"
	"github.com/0xkowalskidev/gamejanitor/internal/service"
	"github.com/0xkowalskidev/gamejanitor/internal/worker"
	"github.com/go-chi/chi/v5"
)

type WorkerHandlers struct {
	registry      *worker.Registry
	settingsSvc   *service.SettingsService
	gameserverSvc *service.GameserverService
	log           *slog.Logger
}

func NewWorkerHandlers(registry *worker.Registry, settingsSvc *service.SettingsService, gameserverSvc *service.GameserverService, log *slog.Logger) *WorkerHandlers {
	return &WorkerHandlers{registry: registry, settingsSvc: settingsSvc, gameserverSvc: gameserverSvc, log: log}
}

type workerAPIView struct {
	ID                string  `json:"id"`
	LanIP             string  `json:"lan_ip"`
	ExternalIP        string  `json:"external_ip"`
	CPUCores          int64   `json:"cpu_cores"`
	MemoryTotalMB     int64   `json:"memory_total_mb"`
	MemoryAvailableMB int64   `json:"memory_available_mb"`
	GameserverCount   int     `json:"gameserver_count"`
	AllocatedMemoryMB int     `json:"allocated_memory_mb"`
	PortRangeStart    *int    `json:"port_range_start"`
	PortRangeEnd      *int    `json:"port_range_end"`
	MaxMemoryMB       *int    `json:"max_memory_mb"`
	MaxGameservers    *int    `json:"max_gameservers"`
	Status            string  `json:"status"`
	LastSeen          *string `json:"last_seen"`
}

func (h *WorkerHandlers) buildWorkerView(info worker.WorkerInfo, gsCount, allocMem int, node *models.WorkerNode) workerAPIView {
	age := time.Since(info.LastSeen)
	status := "stale"
	if age < 15*time.Second {
		status = "connected"
	} else if age < 25*time.Second {
		status = "slow"
	}

	lastSeen := info.LastSeen.UTC().Format(time.RFC3339)

	v := workerAPIView{
		ID:                info.ID,
		LanIP:             info.LanIP,
		ExternalIP:        info.ExternalIP,
		CPUCores:          info.CPUCores,
		MemoryTotalMB:     info.MemoryTotalMB,
		MemoryAvailableMB: info.MemoryAvailableMB,
		GameserverCount:   gsCount,
		AllocatedMemoryMB: allocMem,
		Status:            status,
		LastSeen:          &lastSeen,
	}
	if node != nil {
		v.PortRangeStart = node.PortRangeStart
		v.PortRangeEnd = node.PortRangeEnd
		v.MaxMemoryMB = node.MaxMemoryMB
		v.MaxGameservers = node.MaxGameservers
	}
	return v
}

func (h *WorkerHandlers) nodeStats() (gsCount map[string]int, allocMem map[string]int) {
	gsCount = make(map[string]int)
	allocMem = make(map[string]int)
	gameservers, err := h.gameserverSvc.ListGameservers(models.GameserverFilter{})
	if err != nil {
		h.log.Error("listing gameservers for worker stats", "error", err)
		return
	}
	for _, gs := range gameservers {
		if gs.NodeID != nil && *gs.NodeID != "" {
			gsCount[*gs.NodeID]++
			allocMem[*gs.NodeID] += gs.MemoryLimitMB
		}
	}
	return
}

func (h *WorkerHandlers) List(w http.ResponseWriter, r *http.Request) {
	if h.registry == nil {
		respondOK(w, []workerAPIView{})
		return
	}

	infos := h.registry.ListWorkers()
	gsCount, allocMem := h.nodeStats()

	views := make([]workerAPIView, 0, len(infos))
	for _, info := range infos {
		var node *models.WorkerNode
		if n, err := h.settingsSvc.GetWorkerNode(info.ID); err == nil {
			node = n
		}
		views = append(views, h.buildWorkerView(info, gsCount[info.ID], allocMem[info.ID], node))
	}
	respondOK(w, views)
}

func (h *WorkerHandlers) Get(w http.ResponseWriter, r *http.Request) {
	workerID := chi.URLParam(r, "workerID")

	if h.registry == nil {
		respondError(w, http.StatusNotFound, "multi-node not enabled")
		return
	}

	info, ok := h.registry.GetInfo(workerID)
	if !ok {
		respondError(w, http.StatusNotFound, "worker not found: "+workerID)
		return
	}

	gsCount, allocMem := h.nodeStats()
	var node *models.WorkerNode
	if n, err := h.settingsSvc.GetWorkerNode(workerID); err == nil {
		node = n
	}

	respondOK(w, h.buildWorkerView(info, gsCount[workerID], allocMem[workerID], node))
}
