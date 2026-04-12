package httpapi

import (
	"database/sql"
	"errors"
	"net/http"
	"strings"
	"time"

	"github.com/go-chi/chi/v5"

	"learning-marketplace/internal/app"
	"learning-marketplace/internal/store"
)

type createCohortRequest struct {
	ProductID          string  `json:"product_id"`
	Slug               string  `json:"slug"`
	Title              string  `json:"title"`
	Capacity           int     `json:"capacity"`
	Status             string  `json:"status"`
	StartsAt           string  `json:"starts_at"`
	EndsAt             string  `json:"ends_at"`
	EnrollmentOpensAt  *string `json:"enrollment_opens_at"`
	EnrollmentClosesAt *string `json:"enrollment_closes_at"`
}

type updateCohortRequest struct {
	ProductID          *string `json:"product_id"`
	Slug               *string `json:"slug"`
	Title              *string `json:"title"`
	Capacity           *int    `json:"capacity"`
	Status             *string `json:"status"`
	StartsAt           *string `json:"starts_at"`
	EndsAt             *string `json:"ends_at"`
	EnrollmentOpensAt  *string `json:"enrollment_opens_at"`
	EnrollmentClosesAt *string `json:"enrollment_closes_at"`
}

func registerCohortRoutes(r chi.Router, application *app.App) {
	r.Route("/cohorts", func(r chi.Router) {
		r.Post("/", func(w http.ResponseWriter, r *http.Request) {
			var req createCohortRequest
			if err := decodeJSON(r, &req); err != nil {
				writeError(w, http.StatusBadRequest, err.Error())
				return
			}

			params, err := validateCreateCohort(req)
			if err != nil {
				writeError(w, http.StatusBadRequest, err.Error())
				return
			}

			cohort, err := application.Store.CreateCohort(r.Context(), params)
			if err != nil {
				writeStoreError(w, err)
				return
			}

			writeJSON(w, http.StatusCreated, cohort)
		})

		r.Get("/", func(w http.ResponseWriter, r *http.Request) {
			page := parsePagination(r)
			cohorts, err := application.Store.ListCohorts(r.Context(), store.Pagination{Limit: page.Limit, Offset: page.Offset})
			if err != nil {
				writeStoreError(w, err)
				return
			}

			writeJSON(w, http.StatusOK, map[string]any{"items": cohorts, "limit": page.Limit, "offset": page.Offset})
		})

		r.Get("/{cohortID}", func(w http.ResponseWriter, r *http.Request) {
			cohort, err := application.Store.GetCohort(r.Context(), pathID(r, "cohortID"))
			if err != nil {
				writeStoreError(w, err)
				return
			}

			writeJSON(w, http.StatusOK, cohort)
		})

		r.Patch("/{cohortID}", func(w http.ResponseWriter, r *http.Request) {
			var req updateCohortRequest
			if err := decodeJSON(r, &req); err != nil {
				writeError(w, http.StatusBadRequest, err.Error())
				return
			}

			params, err := validateUpdateCohort(req)
			if err != nil {
				writeError(w, http.StatusBadRequest, err.Error())
				return
			}

			cohort, err := application.Store.UpdateCohort(r.Context(), pathID(r, "cohortID"), params)
			if err != nil {
				writeStoreError(w, err)
				return
			}

			writeJSON(w, http.StatusOK, cohort)
		})

		r.Delete("/{cohortID}", func(w http.ResponseWriter, r *http.Request) {
			if err := application.Store.DeleteCohort(r.Context(), pathID(r, "cohortID")); err != nil {
				writeStoreError(w, err)
				return
			}

			w.WriteHeader(http.StatusNoContent)
		})
	})
}

func validateCreateCohort(req createCohortRequest) (store.CohortCreateParams, error) {
	if err := requireString(req.ProductID, "product_id"); err != nil {
		return store.CohortCreateParams{}, err
	}
	if err := requireString(req.Slug, "slug"); err != nil {
		return store.CohortCreateParams{}, err
	}
	if err := requireString(req.Title, "title"); err != nil {
		return store.CohortCreateParams{}, err
	}
	if req.Capacity <= 0 {
		return store.CohortCreateParams{}, errors.New("capacity must be > 0")
	}
	if !isValidCohortStatus(req.Status) {
		return store.CohortCreateParams{}, errors.New("status must be one of: draft, open, full, closed, cancelled")
	}

	startsAt, err := parseRFC3339(req.StartsAt, "starts_at")
	if err != nil {
		return store.CohortCreateParams{}, err
	}
	endsAt, err := parseRFC3339(req.EndsAt, "ends_at")
	if err != nil {
		return store.CohortCreateParams{}, err
	}
	if !endsAt.After(startsAt) {
		return store.CohortCreateParams{}, errors.New("ends_at must be after starts_at")
	}

	openAt, err := parseOptionalRFC3339(req.EnrollmentOpensAt, "enrollment_opens_at")
	if err != nil {
		return store.CohortCreateParams{}, err
	}
	closeAt, err := parseOptionalRFC3339(req.EnrollmentClosesAt, "enrollment_closes_at")
	if err != nil {
		return store.CohortCreateParams{}, err
	}

	return store.CohortCreateParams{
		ProductID:          strings.TrimSpace(req.ProductID),
		Slug:               strings.TrimSpace(req.Slug),
		Title:              strings.TrimSpace(req.Title),
		Capacity:           req.Capacity,
		Status:             req.Status,
		StartsAt:           startsAt,
		EndsAt:             endsAt,
		EnrollmentOpensAt:  openAt,
		EnrollmentClosesAt: closeAt,
	}, nil
}

func validateUpdateCohort(req updateCohortRequest) (store.CohortUpdateParams, error) {
	params := store.CohortUpdateParams{}
	if req.ProductID != nil {
		trimmed := strings.TrimSpace(*req.ProductID)
		if trimmed == "" {
			return params, errors.New("product_id cannot be empty")
		}
		params.ProductID = &trimmed
	}
	if req.Slug != nil {
		trimmed := strings.TrimSpace(*req.Slug)
		if trimmed == "" {
			return params, errors.New("slug cannot be empty")
		}
		params.Slug = &trimmed
	}
	if req.Title != nil {
		trimmed := strings.TrimSpace(*req.Title)
		if trimmed == "" {
			return params, errors.New("title cannot be empty")
		}
		params.Title = &trimmed
	}
	if req.Capacity != nil {
		if *req.Capacity <= 0 {
			return params, errors.New("capacity must be > 0")
		}
		params.Capacity = req.Capacity
	}
	if req.Status != nil {
		trimmed := strings.TrimSpace(*req.Status)
		if !isValidCohortStatus(trimmed) {
			return params, errors.New("status must be one of: draft, open, full, closed, cancelled")
		}
		params.Status = &trimmed
	}
	if req.StartsAt != nil {
		parsed, err := parseRFC3339(*req.StartsAt, "starts_at")
		if err != nil {
			return params, err
		}
		params.StartsAt = &parsed
	}
	if req.EndsAt != nil {
		parsed, err := parseRFC3339(*req.EndsAt, "ends_at")
		if err != nil {
			return params, err
		}
		params.EndsAt = &parsed
	}
	if req.EnrollmentOpensAt != nil {
		params.UpdateEnrollOpen = true
		if strings.TrimSpace(*req.EnrollmentOpensAt) == "" {
			params.EnrollmentOpensAt = sql.NullTime{}
		} else {
			parsed, err := parseRFC3339(*req.EnrollmentOpensAt, "enrollment_opens_at")
			if err != nil {
				return params, err
			}
			params.EnrollmentOpensAt = sql.NullTime{Time: parsed, Valid: true}
		}
	}
	if req.EnrollmentClosesAt != nil {
		params.UpdateEnrollClose = true
		if strings.TrimSpace(*req.EnrollmentClosesAt) == "" {
			params.EnrollmentClosesAt = sql.NullTime{}
		} else {
			parsed, err := parseRFC3339(*req.EnrollmentClosesAt, "enrollment_closes_at")
			if err != nil {
				return params, err
			}
			params.EnrollmentClosesAt = sql.NullTime{Time: parsed, Valid: true}
		}
	}

	return params, nil
}

func isValidCohortStatus(status string) bool {
	switch status {
	case "draft", "open", "full", "closed", "cancelled":
		return true
	default:
		return false
	}
}

func parseRFC3339(raw, field string) (time.Time, error) {
	trimmed := strings.TrimSpace(raw)
	if trimmed == "" {
		return time.Time{}, errors.New(field + " is required")
	}

	parsed, err := time.Parse(time.RFC3339, trimmed)
	if err != nil {
		return time.Time{}, errors.New(field + " must be RFC3339")
	}

	return parsed, nil
}

func parseOptionalRFC3339(raw *string, field string) (*time.Time, error) {
	if raw == nil {
		return nil, nil
	}

	trimmed := strings.TrimSpace(*raw)
	if trimmed == "" {
		return nil, nil
	}

	parsed, err := time.Parse(time.RFC3339, trimmed)
	if err != nil {
		return nil, errors.New(field + " must be RFC3339")
	}

	return &parsed, nil
}
