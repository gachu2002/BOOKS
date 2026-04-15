package httpapi

import (
	"net/http"

	"github.com/go-chi/chi/v5"

	"learning-marketplace/internal/app"
	"learning-marketplace/internal/store"
)

func registerLibraryRoutes(r chi.Router, application *app.App) {
	r.Route("/users/{userID}/library", func(r chi.Router) {
		r.Get("/", func(w http.ResponseWriter, r *http.Request) {
			page := parsePagination(r)
			source := r.URL.Query().Get("source")
			if source == "" {
				source = "projection"
			}

			userID := pathID(r, "userID")
			switch source {
			case "truth":
				items, selection, err := application.UserShards.ListEntitlementsByUser(r.Context(), userID, store.Pagination{Limit: page.Limit, Offset: page.Offset})
				if err != nil {
					writeStoreError(w, err)
					return
				}
				writeJSON(w, http.StatusOK, map[string]any{
					"items":       items,
					"limit":       page.Limit,
					"offset":      page.Offset,
					"source":      "truth",
					"route":       selection.Route,
					"shard_owner": selection.Owner,
				})
			case "projection":
				selectedStore, selection := application.UserShards.Resolve(userID)
				items, err := selectedStore.ListUserLibrary(r.Context(), userID, store.Pagination{Limit: page.Limit, Offset: page.Offset})
				if err != nil {
					writeStoreError(w, err)
					return
				}
				freshness, err := selectedStore.GetUserLibraryProjectionStatus(r.Context(), userID)
				if err != nil {
					writeStoreError(w, err)
					return
				}
				writeJSON(w, http.StatusOK, map[string]any{
					"items":       items,
					"limit":       page.Limit,
					"offset":      page.Offset,
					"source":      "projection",
					"freshness":   freshness,
					"route":       selection.Route,
					"shard_owner": selection.Owner,
				})
			default:
				writeError(w, http.StatusBadRequest, "source must be one of: truth, projection")
			}
		})
	})
}
