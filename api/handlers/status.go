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

type gameserverOverview struct {
	ID            string  `json:"id"`
	Name          string  `json:"name"`
	GameID        string  `json:"game_id"`
	Status        string  `json:"status"`
	MemoryUsageMB int     `json:"memory_usage_mb"`
	MemoryLimitMB int     `json:"memory_limit_mb"`
	CPUPercent    float64 `json:"cpu_percent"`
	PlayersOnline *int    `json:"players_online"`
	MaxPlayers    *int    `json:"max_players"`
}

type statusSummary struct {
	Total   int `json:"total"`
	Running int `json:"running"`
	Stopped int `json:"stopped"`
}

type clusterSummary struct {
	Workers           int     `json:"workers"`
	WorkersCordoned   int     `json:"workers_cordoned"`
	TotalMemoryMB     int64   `json:"total_memory_mb"`
	AllocatedMemoryMB int     `json:"allocated_memory_mb"`
	TotalCPU          float64 `json:"total_cpu"`
	AllocatedCPU      float64 `json:"allocated_cpu"`
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

	summary := statusSummary{Total: len(gameservers)}
	overviews := make([]gameserverOverview, 0, len(gameservers))

	for _, gs := range gameservers {
		o := gameserverOverview{
			ID:            gs.ID,
			Name:          gs.Name,
			GameID:        gs.GameID,
			Status:        gs.Status,
			MemoryLimitMB: gs.MemoryLimitMB,
		}

		isRunning := gs.Status == service.StatusStarted || gs.Status == service.StatusRunning

		if isRunning {
			summary.Running++

			if gs.ContainerID != nil {
				stats, err := h.gameserverSvc.GetGameserverStats(r.Context(), gs.ID)
				if err != nil {
					h.log.Warn("failed to get gameserver stats", "id", gs.ID, "error", err)
				} else {
					o.MemoryUsageMB = stats.MemoryUsageMB
					o.CPUPercent = stats.CPUPercent
				}
			}

			if qd := h.querySvc.GetQueryData(gs.ID); qd != nil {
				o.PlayersOnline = &qd.PlayersOnline
				o.MaxPlayers = &qd.MaxPlayers
			}
		} else if gs.Status == service.StatusStopped {
			summary.Stopped++
		}

		overviews = append(overviews, o)
	}

	cluster := h.buildClusterSummary(gameservers)

	respondOK(w, map[string]any{
		"gameservers": overviews,
		"summary":     summary,
		"cluster":     cluster,
	})
}

func (h *StatusHandlers) buildClusterSummary(gameservers []models.Gameserver) clusterSummary {
	var cluster clusterSummary

	workers, err := h.workerSvc.List()
	if err != nil {
		h.log.Error("listing workers for cluster summary", "error", err)
		return cluster
	}

	cluster.Workers = len(workers)
	for _, w := range workers {
		cluster.TotalMemoryMB += w.MemoryTotalMB
		cluster.TotalCPU += float64(w.CPUCores)
		if w.Cordoned {
			cluster.WorkersCordoned++
		}
	}

	for _, gs := range gameservers {
		cluster.AllocatedMemoryMB += gs.MemoryLimitMB
		cluster.AllocatedCPU += gs.CPULimit
	}

	return cluster
}
