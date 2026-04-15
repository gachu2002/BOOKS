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
			userID := pathID(r, "userID")

			source := "primary"
			var (
				items     []store.EntitlementWithProduct
				selection store.UserShardSelection
				err       error
			)
			if consistency == "eventual" && application.ReaderStore != nil {
				source = "replica"
				items, err = application.ReaderStore.ListEntitlementsByUser(r.Context(), userID, store.Pagination{Limit: page.Limit, Offset: page.Offset})
				selection = store.UserShardSelection{Route: application.ReaderStore.RouteUser(userID), Owner: "replica"}
			} else {
				items, selection, err = application.UserShards.ListEntitlementsByUser(r.Context(), userID, store.Pagination{Limit: page.Limit, Offset: page.Offset})
			}
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
				"route":       selection.Route,
				"shard_owner": selection.Owner,
			})
		})
	})
}
