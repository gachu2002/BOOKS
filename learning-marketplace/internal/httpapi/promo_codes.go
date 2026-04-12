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

type createPromoCodeRequest struct {
	Code           string  `json:"code"`
	DiscountType   string  `json:"discount_type"`
	DiscountValue  int     `json:"discount_value"`
	MaxRedemptions *int    `json:"max_redemptions"`
	StartsAt       *string `json:"starts_at"`
	ExpiresAt      *string `json:"expires_at"`
}

type updatePromoCodeRequest struct {
	Code           *string `json:"code"`
	DiscountType   *string `json:"discount_type"`
	DiscountValue  *int    `json:"discount_value"`
	MaxRedemptions *int    `json:"max_redemptions"`
	StartsAt       *string `json:"starts_at"`
	ExpiresAt      *string `json:"expires_at"`
}

func registerPromoCodeRoutes(r chi.Router, application *app.App) {
	r.Route("/promo-codes", func(r chi.Router) {
		r.Post("/", func(w http.ResponseWriter, r *http.Request) {
			var req createPromoCodeRequest
			if err := decodeJSON(r, &req); err != nil {
				writeError(w, http.StatusBadRequest, err.Error())
				return
			}

			params, err := validateCreatePromoCode(req)
			if err != nil {
				writeError(w, http.StatusBadRequest, err.Error())
				return
			}

			promo, err := application.Store.CreatePromoCode(r.Context(), params)
			if err != nil {
				writeStoreError(w, err)
				return
			}

			writeJSON(w, http.StatusCreated, promo)
		})

		r.Get("/", func(w http.ResponseWriter, r *http.Request) {
			page := parsePagination(r)
			promos, err := application.Store.ListPromoCodes(r.Context(), store.Pagination{Limit: page.Limit, Offset: page.Offset})
			if err != nil {
				writeStoreError(w, err)
				return
			}

			writeJSON(w, http.StatusOK, map[string]any{"items": promos, "limit": page.Limit, "offset": page.Offset})
		})

		r.Get("/{promoCodeID}", func(w http.ResponseWriter, r *http.Request) {
			promo, err := application.Store.GetPromoCode(r.Context(), pathID(r, "promoCodeID"))
			if err != nil {
				writeStoreError(w, err)
				return
			}

			writeJSON(w, http.StatusOK, promo)
		})

		r.Patch("/{promoCodeID}", func(w http.ResponseWriter, r *http.Request) {
			var req updatePromoCodeRequest
			if err := decodeJSON(r, &req); err != nil {
				writeError(w, http.StatusBadRequest, err.Error())
				return
			}

			params, err := validateUpdatePromoCode(req)
			if err != nil {
				writeError(w, http.StatusBadRequest, err.Error())
				return
			}

			promo, err := application.Store.UpdatePromoCode(r.Context(), pathID(r, "promoCodeID"), params)
			if err != nil {
				writeStoreError(w, err)
				return
			}

			writeJSON(w, http.StatusOK, promo)
		})

		r.Delete("/{promoCodeID}", func(w http.ResponseWriter, r *http.Request) {
			if err := application.Store.DeletePromoCode(r.Context(), pathID(r, "promoCodeID")); err != nil {
				writeStoreError(w, err)
				return
			}

			w.WriteHeader(http.StatusNoContent)
		})
	})
}

func validateCreatePromoCode(req createPromoCodeRequest) (store.PromoCodeCreateParams, error) {
	if err := requireString(req.Code, "code"); err != nil {
		return store.PromoCodeCreateParams{}, err
	}
	if req.DiscountType != "percent" && req.DiscountType != "fixed" {
		return store.PromoCodeCreateParams{}, errors.New("discount_type must be one of: percent, fixed")
	}
	if req.DiscountValue <= 0 {
		return store.PromoCodeCreateParams{}, errors.New("discount_value must be > 0")
	}
	if req.MaxRedemptions != nil && *req.MaxRedemptions <= 0 {
		return store.PromoCodeCreateParams{}, errors.New("max_redemptions must be > 0 when provided")
	}

	startsAt, err := parseOptionalRFC3339(req.StartsAt, "starts_at")
	if err != nil {
		return store.PromoCodeCreateParams{}, err
	}
	expiresAt, err := parseOptionalRFC3339(req.ExpiresAt, "expires_at")
	if err != nil {
		return store.PromoCodeCreateParams{}, err
	}
	if startsAt != nil && expiresAt != nil && !expiresAt.After(*startsAt) {
		return store.PromoCodeCreateParams{}, errors.New("expires_at must be after starts_at")
	}

	return store.PromoCodeCreateParams{
		Code:           strings.TrimSpace(req.Code),
		DiscountType:   req.DiscountType,
		DiscountValue:  req.DiscountValue,
		MaxRedemptions: req.MaxRedemptions,
		StartsAt:       startsAt,
		ExpiresAt:      expiresAt,
	}, nil
}

func validateUpdatePromoCode(req updatePromoCodeRequest) (store.PromoCodeUpdateParams, error) {
	params := store.PromoCodeUpdateParams{}
	if req.Code != nil {
		trimmed := strings.TrimSpace(*req.Code)
		if trimmed == "" {
			return params, errors.New("code cannot be empty")
		}
		params.Code = &trimmed
	}
	if req.DiscountType != nil {
		trimmed := strings.TrimSpace(*req.DiscountType)
		if trimmed != "percent" && trimmed != "fixed" {
			return params, errors.New("discount_type must be one of: percent, fixed")
		}
		params.DiscountType = &trimmed
	}
	if req.DiscountValue != nil {
		if *req.DiscountValue <= 0 {
			return params, errors.New("discount_value must be > 0")
		}
		params.DiscountValue = req.DiscountValue
	}
	if req.MaxRedemptions != nil {
		params.UpdateMaxRedemptions = true
		if *req.MaxRedemptions <= 0 {
			return params, errors.New("max_redemptions must be > 0 when provided")
		}
		params.MaxRedemptions = sql.NullInt32{Int32: int32(*req.MaxRedemptions), Valid: true}
	}
	if req.StartsAt != nil {
		params.UpdateStartsAt = true
		if strings.TrimSpace(*req.StartsAt) == "" {
			params.StartsAt = sql.NullTime{}
		} else {
			parsed, err := time.Parse(time.RFC3339, strings.TrimSpace(*req.StartsAt))
			if err != nil {
				return params, errors.New("starts_at must be RFC3339")
			}
			params.StartsAt = sql.NullTime{Time: parsed, Valid: true}
		}
	}
	if req.ExpiresAt != nil {
		params.UpdateExpiresAt = true
		if strings.TrimSpace(*req.ExpiresAt) == "" {
			params.ExpiresAt = sql.NullTime{}
		} else {
			parsed, err := time.Parse(time.RFC3339, strings.TrimSpace(*req.ExpiresAt))
			if err != nil {
				return params, errors.New("expires_at must be RFC3339")
			}
			params.ExpiresAt = sql.NullTime{Time: parsed, Valid: true}
		}
	}

	if params.UpdateStartsAt && params.UpdateExpiresAt && params.StartsAt.Valid && params.ExpiresAt.Valid && !params.ExpiresAt.Time.After(params.StartsAt.Time) {
		return params, errors.New("expires_at must be after starts_at")
	}

	return params, nil
}
