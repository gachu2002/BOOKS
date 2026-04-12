package httpapi

import (
	"errors"
	"net/http"
	"strings"

	"github.com/go-chi/chi/v5"

	"learning-marketplace/internal/app"
	"learning-marketplace/internal/store"
)

type checkoutCohortRequest struct {
	UserID            string  `json:"user_id"`
	CohortID          string  `json:"cohort_id"`
	PromoCode         *string `json:"promo_code"`
	PaymentProvider   string  `json:"payment_provider"`
	IdempotencyKey    string  `json:"idempotency_key"`
	ProviderPaymentID *string `json:"provider_payment_id"`
}

func registerCheckoutRoutes(r chi.Router, application *app.App) {
	r.Route("/checkouts", func(r chi.Router) {
		r.Post("/live-cohorts", func(w http.ResponseWriter, r *http.Request) {
			var req checkoutCohortRequest
			if err := decodeJSON(r, &req); err != nil {
				writeError(w, http.StatusBadRequest, err.Error())
				return
			}

			params, err := validateCheckoutCohort(req)
			if err != nil {
				writeError(w, http.StatusBadRequest, err.Error())
				return
			}

			result, err := application.Store.CheckoutCohort(r.Context(), params)
			if err != nil {
				writeCheckoutError(w, err)
				return
			}

			status := http.StatusCreated
			if result.IdempotentReplay {
				status = http.StatusOK
			}

			writeJSON(w, status, result)
		})
	})
}

func validateCheckoutCohort(req checkoutCohortRequest) (store.CheckoutParams, error) {
	if err := requireString(req.UserID, "user_id"); err != nil {
		return store.CheckoutParams{}, err
	}
	if err := requireString(req.CohortID, "cohort_id"); err != nil {
		return store.CheckoutParams{}, err
	}
	if err := requireString(req.PaymentProvider, "payment_provider"); err != nil {
		return store.CheckoutParams{}, err
	}
	if err := requireString(req.IdempotencyKey, "idempotency_key"); err != nil {
		return store.CheckoutParams{}, err
	}

	params := store.CheckoutParams{
		UserID:            strings.TrimSpace(req.UserID),
		CohortID:          strings.TrimSpace(req.CohortID),
		PaymentProvider:   strings.TrimSpace(req.PaymentProvider),
		IdempotencyKey:    strings.TrimSpace(req.IdempotencyKey),
		ProviderPaymentID: trimOptionalString(req.ProviderPaymentID),
	}

	if req.PromoCode != nil {
		trimmed := strings.TrimSpace(*req.PromoCode)
		if trimmed != "" {
			params.PromoCode = &trimmed
		}
	}

	return params, nil
}

func writeCheckoutError(w http.ResponseWriter, err error) {
	switch {
	case errors.Is(err, store.ErrSoldOut):
		writeError(w, http.StatusConflict, "cohort is sold out")
	case errors.Is(err, store.ErrCohortNotOpen):
		writeError(w, http.StatusConflict, "cohort is not open for checkout")
	case errors.Is(err, store.ErrPromoCodeInvalid):
		writeError(w, http.StatusBadRequest, "promo code is invalid")
	case errors.Is(err, store.ErrPromoCodeExhausted):
		writeError(w, http.StatusConflict, "promo code has reached its redemption limit")
	default:
		writeStoreError(w, err)
	}
}

func trimOptionalString(value *string) *string {
	if value == nil {
		return nil
	}

	trimmed := strings.TrimSpace(*value)
	if trimmed == "" {
		return nil
	}

	return &trimmed
}
