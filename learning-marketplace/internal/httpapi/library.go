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
				items, err := application.Store.ListEntitlementsByUser(r.Context(), userID, store.Pagination{Limit: page.Limit, Offset: page.Offset})
				if err != nil {
					writeStoreError(w, err)
					return
				}
				writeJSON(w, http.StatusOK, map[string]any{
					"items":  items,
					"limit":  page.Limit,
					"offset": page.Offset,
					"source": "truth",
				})
			case "projection":
				items, err := application.Store.ListUserLibrary(r.Context(), userID, store.Pagination{Limit: page.Limit, Offset: page.Offset})
				if err != nil {
					writeStoreError(w, err)
					return
				}
				writeJSON(w, http.StatusOK, map[string]any{
					"items":  items,
					"limit":  page.Limit,
					"offset": page.Offset,
					"source": "projection",
				})
			default:
				writeError(w, http.StatusBadRequest, "source must be one of: truth, projection")
			}
		})
	})
}
