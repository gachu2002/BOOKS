package httpapi

import (
	"context"
	"encoding/json"
	"net/http"
	"time"

	"learning-marketplace/internal/app"
	"learning-marketplace/internal/observability"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"github.com/prometheus/client_golang/prometheus/promhttp"
)

func NewRouter(application *app.App) http.Handler {
	r := chi.NewRouter()

	// Global middleware keeps request IDs, timing, recovery, and metrics consistent.
	r.Use(middleware.RequestID)
	r.Use(middleware.RealIP)
	r.Use(middleware.Recoverer)
	r.Use(middleware.Timeout(30 * time.Second))
	r.Use(observability.InstrumentHTTP)
	r.Use(observability.LogRequests)

	// Operational endpoints stay outside the versioned API surface.
	r.Handle("/metrics", promhttp.Handler())

	r.Get("/healthz", func(w http.ResponseWriter, r *http.Request) {
		writeJSON(w, http.StatusOK, map[string]any{
			"status": "ok",
			"time":   time.Now().UTC().Format(time.RFC3339),
		})
	})

	r.Get("/readyz", func(w http.ResponseWriter, r *http.Request) {
		ctx, cancel := context.WithTimeout(r.Context(), 2*time.Second)
		defer cancel()

		if err := application.DB.PingContext(ctx); err != nil {
			writeJSON(w, http.StatusServiceUnavailable, map[string]any{
				"status": "not_ready",
				"error":  err.Error(),
			})
			return
		}

		writeJSON(w, http.StatusOK, map[string]any{
			"status": "ready",
			"env":    application.Config.AppEnv,
		})
	})

	// Versioned application routes live under /v1.
	r.Route("/v1", func(r chi.Router) {
		r.Get("/meta", func(w http.ResponseWriter, r *http.Request) {
			writeJSON(w, http.StatusOK, map[string]any{
				"service": "learning-marketplace",
				"env":     application.Config.AppEnv,
				"db":      application.Config.Postgres.DB,
				"phase":   "phase-11-batch-and-projection-foundation",
			})
		})

		registerUserRoutes(r, application)
		registerProductRoutes(r, application)
		registerCohortRoutes(r, application)
		registerPromoCodeRoutes(r, application)
		registerCheckoutRoutes(r, application)
		registerEntitlementRoutes(r, application)
		registerLibraryRoutes(r, application)
		registerCoordinationRoutes(r, application)
		registerReportRoutes(r, application)
	})

	return r
}

func writeJSON(w http.ResponseWriter, status int, payload any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)

	_ = json.NewEncoder(w).Encode(payload)
}
