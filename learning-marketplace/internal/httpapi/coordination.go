package httpapi

import (
	"context"
	"errors"
	"net/http"
	"strings"
	"time"

	"github.com/go-chi/chi/v5"

	"learning-marketplace/internal/app"
	"learning-marketplace/internal/coordination"
	"learning-marketplace/internal/store"
)

type acquireLeaseRequest struct {
	Holder     string `json:"holder"`
	TTLSeconds int64  `json:"ttl_seconds"`
}

type releaseLeaseRequest struct {
	LeaseID int64 `json:"lease_id"`
}

type fencedIncrementRequest struct {
	Holder       string `json:"holder"`
	FencingToken int64  `json:"fencing_token"`
}

func registerCoordinationRoutes(r chi.Router, application *app.App) {
	r.Route("/lease-lab/counters/{resource}", func(r chi.Router) {
		r.Post("/acquire", func(w http.ResponseWriter, r *http.Request) {
			if application.LeaseStore == nil {
				writeError(w, http.StatusServiceUnavailable, "lease store is not configured")
				return
			}

			var req acquireLeaseRequest
			if err := decodeJSON(r, &req); err != nil {
				writeError(w, http.StatusBadRequest, err.Error())
				return
			}
			if strings.TrimSpace(req.Holder) == "" {
				writeError(w, http.StatusBadRequest, "holder is required")
				return
			}
			if req.TTLSeconds <= 0 {
				req.TTLSeconds = 15
			}

			ctx, cancel := context.WithTimeout(r.Context(), 5*time.Second)
			defer cancel()
			grant, err := application.LeaseStore.Acquire(ctx, pathID(r, "resource"), strings.TrimSpace(req.Holder), req.TTLSeconds)
			if err != nil {
				if errors.Is(err, coordination.ErrLeaseAlreadyHeld) {
					writeError(w, http.StatusConflict, "lease is already held")
					return
				}
				writeError(w, http.StatusInternalServerError, "failed to acquire lease")
				return
			}

			writeJSON(w, http.StatusCreated, grant)
		})

		r.Post("/release", func(w http.ResponseWriter, r *http.Request) {
			if application.LeaseStore == nil {
				writeError(w, http.StatusServiceUnavailable, "lease store is not configured")
				return
			}

			var req releaseLeaseRequest
			if err := decodeJSON(r, &req); err != nil {
				writeError(w, http.StatusBadRequest, err.Error())
				return
			}

			ctx, cancel := context.WithTimeout(r.Context(), 5*time.Second)
			defer cancel()
			if err := application.LeaseStore.Release(ctx, req.LeaseID); err != nil {
				writeError(w, http.StatusInternalServerError, "failed to release lease")
				return
			}

			w.WriteHeader(http.StatusNoContent)
		})

		r.Post("/increment", func(w http.ResponseWriter, r *http.Request) {
			var req fencedIncrementRequest
			if err := decodeJSON(r, &req); err != nil {
				writeError(w, http.StatusBadRequest, err.Error())
				return
			}
			if strings.TrimSpace(req.Holder) == "" {
				writeError(w, http.StatusBadRequest, "holder is required")
				return
			}
			if req.FencingToken <= 0 {
				writeError(w, http.StatusBadRequest, "fencing_token must be > 0")
				return
			}

			counter, err := application.Store.ApplyFencedIncrement(r.Context(), pathID(r, "resource"), strings.TrimSpace(req.Holder), req.FencingToken)
			if err != nil {
				if errors.Is(err, store.ErrStaleFencingToken) {
					writeError(w, http.StatusConflict, "stale fencing token")
					return
				}
				writeStoreError(w, err)
				return
			}

			writeJSON(w, http.StatusOK, counter)
		})

		r.Get("/", func(w http.ResponseWriter, r *http.Request) {
			counter, err := application.Store.GetProtectedCounter(r.Context(), pathID(r, "resource"))
			if err != nil {
				if errors.Is(err, store.ErrNotFound) {
					response := map[string]any{"resource_name": pathID(r, "resource"), "value": 0, "last_fencing_token": 0}
					if application.LeaseStore != nil {
						ctx, cancel := context.WithTimeout(r.Context(), 5*time.Second)
						defer cancel()
						lease, leaseErr := application.LeaseStore.CurrentHolder(ctx, pathID(r, "resource"))
						if leaseErr == nil {
							response["lease"] = lease
						}
					}
					writeJSON(w, http.StatusOK, response)
					return
				}
				writeStoreError(w, err)
				return
			}

			response := map[string]any{"counter": counter}
			if application.LeaseStore != nil {
				ctx, cancel := context.WithTimeout(r.Context(), 5*time.Second)
				defer cancel()
				lease, leaseErr := application.LeaseStore.CurrentHolder(ctx, pathID(r, "resource"))
				if leaseErr == nil {
					response["lease"] = lease
				}
			}

			writeJSON(w, http.StatusOK, response)
		})
	})
}
