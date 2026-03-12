package web

import (
	"fmt"
	"log/slog"
	"net/http"

	"github.com/0xkowalskidev/gamejanitor/internal/docker"
	"github.com/0xkowalskidev/gamejanitor/internal/service"
	"github.com/0xkowalskidev/gamejanitor/internal/web/handlers"
	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
)

func NewRouter(
	gameSvc *service.GameService,
	gameserverSvc *service.GameserverService,
	dockerClient *docker.Client,
	broadcaster *service.EventBroadcaster,
	log *slog.Logger,
) http.Handler {
	r := chi.NewRouter()

	r.Use(middleware.Recoverer)
	r.Use(middleware.RequestID)

	r.Get("/health", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		fmt.Fprint(w, "ok")
	})

	gameHandlers := handlers.NewGameHandlers(gameSvc, log)
	gameserverHandlers := handlers.NewGameserverHandlers(gameserverSvc, dockerClient, log)
	eventHandlers := handlers.NewEventHandlers(broadcaster, log)

	r.Route("/api", func(r chi.Router) {
		r.Use(jsonContentType)

		r.Route("/games", func(r chi.Router) {
			r.Get("/", gameHandlers.List)
			r.Post("/", gameHandlers.Create)
			r.Route("/{id}", func(r chi.Router) {
				r.Get("/", gameHandlers.Get)
				r.Put("/", gameHandlers.Update)
				r.Delete("/", gameHandlers.Delete)
			})
		})

		r.Route("/gameservers", func(r chi.Router) {
			r.Get("/", gameserverHandlers.List)
			r.Post("/", gameserverHandlers.Create)
			r.Route("/{id}", func(r chi.Router) {
				r.Get("/", gameserverHandlers.Get)
				r.Put("/", gameserverHandlers.Update)
				r.Delete("/", gameserverHandlers.Delete)
				r.Post("/start", gameserverHandlers.Start)
				r.Post("/stop", gameserverHandlers.Stop)
				r.Post("/restart", gameserverHandlers.Restart)
				r.Post("/update-game", gameserverHandlers.UpdateServerGame)
				r.Post("/reinstall", gameserverHandlers.Reinstall)
				r.Get("/status", gameserverHandlers.Status)
			})
		})

		r.Get("/events", eventHandlers.SSE)
	})

	return r
}

func jsonContentType(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		next.ServeHTTP(w, r)
	})
}
