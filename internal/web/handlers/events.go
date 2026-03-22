package handlers

import (
	"encoding/json"
	"fmt"
	"log/slog"
	"net/http"
	"time"

	"github.com/warsmite/gamejanitor/internal/service"
)

type EventHandlers struct {
	broadcaster *service.EventBus
	log         *slog.Logger
}

func NewEventHandlers(broadcaster *service.EventBus, log *slog.Logger) *EventHandlers {
	return &EventHandlers{broadcaster: broadcaster, log: log}
}

func (h *EventHandlers) SSE(w http.ResponseWriter, r *http.Request) {
	flusher, ok := w.(http.Flusher)
	if !ok {
		http.Error(w, "streaming not supported", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "text/event-stream")
	w.Header().Set("Cache-Control", "no-cache")
	w.Header().Set("Connection", "keep-alive")

	ch, unsubscribe := h.broadcaster.Subscribe()
	defer unsubscribe()

	// Initial heartbeat to establish connection
	fmt.Fprint(w, ": heartbeat\n\n")
	flusher.Flush()

	heartbeat := time.NewTicker(30 * time.Second)
	defer heartbeat.Stop()

	for {
		select {
		case <-r.Context().Done():
			return
		case event, ok := <-ch:
			if !ok {
				return
			}
			statusEvent, isStatus := event.(service.StatusEvent)
			if !isStatus {
				continue
			}
			data, err := json.Marshal(statusEvent)
			if err != nil {
				h.log.Error("marshaling SSE event", "error", err)
				continue
			}
			fmt.Fprintf(w, "event: status\ndata: %s\n\n", data)
			flusher.Flush()
		case <-heartbeat.C:
			fmt.Fprint(w, ": heartbeat\n\n")
			flusher.Flush()
		}
	}
}
