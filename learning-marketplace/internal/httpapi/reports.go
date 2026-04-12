package httpapi

import (
	"net/http"

	"github.com/go-chi/chi/v5"

	"learning-marketplace/internal/app"
)

func registerReportRoutes(r chi.Router, application *app.App) {
	r.Route("/reports", func(r chi.Router) {
		r.Get("/daily-revenue", func(w http.ResponseWriter, r *http.Request) {
			page := parsePagination(r)
			items, err := application.Reporter.ListDailyRevenue(r.Context(), page.Limit, page.Offset)
			if err != nil {
				writeError(w, http.StatusInternalServerError, "failed to list daily revenue report")
				return
			}

			writeJSON(w, http.StatusOK, map[string]any{"items": items, "limit": page.Limit, "offset": page.Offset})
		})

		r.Get("/cohort-fill", func(w http.ResponseWriter, r *http.Request) {
			page := parsePagination(r)
			items, err := application.Reporter.ListCohortFill(r.Context(), page.Limit, page.Offset)
			if err != nil {
				writeError(w, http.StatusInternalServerError, "failed to list cohort fill report")
				return
			}

			writeJSON(w, http.StatusOK, map[string]any{"items": items, "limit": page.Limit, "offset": page.Offset})
		})
	})
}
