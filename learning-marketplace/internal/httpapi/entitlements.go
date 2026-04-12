package httpapi

import (
	"net/http"

	"github.com/go-chi/chi/v5"

	"learning-marketplace/internal/app"
	"learning-marketplace/internal/store"
)

func registerEntitlementRoutes(r chi.Router, application *app.App) {
	r.Route("/users/{userID}/entitlements", func(r chi.Router) {
		r.Get("/", func(w http.ResponseWriter, r *http.Request) {
			page := parsePagination(r)
			consistency := r.URL.Query().Get("consistency")
			if consistency == "" {
				consistency = "strong"
			}

			selectedStore := application.Store
			source := "primary"
			if consistency == "eventual" && application.ReaderStore != nil {
				selectedStore = application.ReaderStore
				source = "replica"
			}

			items, err := selectedStore.ListEntitlementsByUser(r.Context(), pathID(r, "userID"), store.Pagination{Limit: page.Limit, Offset: page.Offset})
			if err != nil {
				writeStoreError(w, err)
				return
			}

			writeJSON(w, http.StatusOK, map[string]any{
				"items":       items,
				"limit":       page.Limit,
				"offset":      page.Offset,
				"consistency": consistency,
				"source":      source,
			})
		})
	})
}
