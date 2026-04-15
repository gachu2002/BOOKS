package httpapi

import (
	"encoding/base64"
	"encoding/json"
	"errors"
	"net/http"
	"strings"
	"time"

	"github.com/go-chi/chi/v5"

	"learning-marketplace/internal/app"
	"learning-marketplace/internal/store"
)

type createProductRequest struct {
	Slug        string          `json:"slug"`
	Title       string          `json:"title"`
	Description string          `json:"description"`
	ProductType string          `json:"product_type"`
	PriceCents  int             `json:"price_cents"`
	Currency    string          `json:"currency"`
	Published   bool            `json:"published"`
	Metadata    json.RawMessage `json:"metadata"`
}

type updateProductRequest struct {
	Slug        *string          `json:"slug"`
	Title       *string          `json:"title"`
	Description *string          `json:"description"`
	ProductType *string          `json:"product_type"`
	PriceCents  *int             `json:"price_cents"`
	Currency    *string          `json:"currency"`
	Published   *bool            `json:"published"`
	Metadata    *json.RawMessage `json:"metadata"`
}

func registerProductRoutes(r chi.Router, application *app.App) {
	r.Route("/products", func(r chi.Router) {
		r.Post("/", func(w http.ResponseWriter, r *http.Request) {
			var req createProductRequest
			if err := decodeJSON(r, &req); err != nil {
				writeError(w, http.StatusBadRequest, err.Error())
				return
			}

			params, err := validateCreateProduct(req)
			if err != nil {
				writeError(w, http.StatusBadRequest, err.Error())
				return
			}

			product, err := application.Store.CreateProduct(r.Context(), params)
			if err != nil {
				writeStoreError(w, err)
				return
			}

			writeJSON(w, http.StatusCreated, product)
		})

		r.Get("/", func(w http.ResponseWriter, r *http.Request) {
			page := parsePagination(r)
			products, err := application.Store.ListProducts(r.Context(), store.Pagination{Limit: page.Limit, Offset: page.Offset})
			if err != nil {
				writeStoreError(w, err)
				return
			}

			writeJSON(w, http.StatusOK, map[string]any{"items": products, "limit": page.Limit, "offset": page.Offset})
		})

		r.Get("/published", func(w http.ResponseWriter, r *http.Request) {
			page := parsePagination(r)
			cursor, err := decodeProductCursor(r.URL.Query().Get("after"))
			if err != nil {
				writeError(w, http.StatusBadRequest, err.Error())
				return
			}

			products, err := application.Store.ListPublishedProductsByCursor(r.Context(), page.Limit, cursor)
			if err != nil {
				writeStoreError(w, err)
				return
			}

			response := map[string]any{"items": products, "limit": page.Limit}
			if len(products) > 0 {
				last := products[len(products)-1]
				response["next_cursor"] = encodeProductCursor(store.ProductCursor{CreatedAt: last.CreatedAt, ID: last.ID})
			}

			writeJSON(w, http.StatusOK, response)
		})

		r.Get("/search", func(w http.ResponseWriter, r *http.Request) {
			page := parsePagination(r)
			searchQuery := strings.TrimSpace(r.URL.Query().Get("q"))
			if searchQuery == "" {
				writeError(w, http.StatusBadRequest, "q is required")
				return
			}

			results, err := application.Store.SearchPublishedProducts(r.Context(), searchQuery, page.Limit)
			if err != nil {
				writeStoreError(w, err)
				return
			}

			writeJSON(w, http.StatusOK, map[string]any{"items": results, "limit": page.Limit, "query": searchQuery})
		})

		r.Get("/{productID}", func(w http.ResponseWriter, r *http.Request) {
			product, err := application.Store.GetProduct(r.Context(), pathID(r, "productID"))
			if err != nil {
				writeStoreError(w, err)
				return
			}

			writeJSON(w, http.StatusOK, product)
		})

		r.Patch("/{productID}", func(w http.ResponseWriter, r *http.Request) {
			var req updateProductRequest
			if err := decodeJSON(r, &req); err != nil {
				writeError(w, http.StatusBadRequest, err.Error())
				return
			}

			params, err := validateUpdateProduct(req)
			if err != nil {
				writeError(w, http.StatusBadRequest, err.Error())
				return
			}

			product, err := application.Store.UpdateProduct(r.Context(), pathID(r, "productID"), params)
			if err != nil {
				writeStoreError(w, err)
				return
			}

			writeJSON(w, http.StatusOK, product)
		})

		r.Delete("/{productID}", func(w http.ResponseWriter, r *http.Request) {
			if err := application.Store.DeleteProduct(r.Context(), pathID(r, "productID")); err != nil {
				writeStoreError(w, err)
				return
			}

			w.WriteHeader(http.StatusNoContent)
		})
	})
}

func encodeProductCursor(cursor store.ProductCursor) string {
	payload := cursor.CreatedAt.UTC().Format(time.RFC3339Nano) + "|" + cursor.ID
	return base64.RawURLEncoding.EncodeToString([]byte(payload))
}

func decodeProductCursor(raw string) (*store.ProductCursor, error) {
	trimmed := strings.TrimSpace(raw)
	if trimmed == "" {
		return nil, nil
	}

	decoded, err := base64.RawURLEncoding.DecodeString(trimmed)
	if err != nil {
		return nil, errors.New("after cursor must be valid base64url")
	}

	parts := strings.SplitN(string(decoded), "|", 2)
	if len(parts) != 2 {
		return nil, errors.New("after cursor must contain timestamp and id")
	}

	createdAt, err := time.Parse(time.RFC3339Nano, parts[0])
	if err != nil {
		return nil, errors.New("after cursor timestamp must be RFC3339Nano")
	}

	if strings.TrimSpace(parts[1]) == "" {
		return nil, errors.New("after cursor id cannot be empty")
	}

	return &store.ProductCursor{CreatedAt: createdAt, ID: parts[1]}, nil
}

func validateCreateProduct(req createProductRequest) (store.ProductCreateParams, error) {
	if err := requireString(req.Slug, "slug"); err != nil {
		return store.ProductCreateParams{}, err
	}
	if err := requireString(req.Title, "title"); err != nil {
		return store.ProductCreateParams{}, err
	}
	if req.ProductType != "digital_download" && req.ProductType != "live_cohort" {
		return store.ProductCreateParams{}, errors.New("product_type must be one of: digital_download, live_cohort")
	}
	if req.PriceCents < 0 {
		return store.ProductCreateParams{}, errors.New("price_cents must be >= 0")
	}

	currency := strings.TrimSpace(req.Currency)
	if currency == "" {
		currency = "USD"
	}

	return store.ProductCreateParams{
		Slug:        strings.TrimSpace(req.Slug),
		Title:       strings.TrimSpace(req.Title),
		Description: req.Description,
		ProductType: req.ProductType,
		PriceCents:  req.PriceCents,
		Currency:    currency,
		Published:   req.Published,
		Metadata:    req.Metadata,
	}, nil
}

func validateUpdateProduct(req updateProductRequest) (store.ProductUpdateParams, error) {
	params := store.ProductUpdateParams{}
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
	if req.Description != nil {
		params.Description = req.Description
	}
	if req.ProductType != nil {
		trimmed := strings.TrimSpace(*req.ProductType)
		if trimmed != "digital_download" && trimmed != "live_cohort" {
			return params, errors.New("product_type must be one of: digital_download, live_cohort")
		}
		params.ProductType = &trimmed
	}
	if req.PriceCents != nil {
		if *req.PriceCents < 0 {
			return params, errors.New("price_cents must be >= 0")
		}
		params.PriceCents = req.PriceCents
	}
	if req.Currency != nil {
		trimmed := strings.TrimSpace(*req.Currency)
		if trimmed == "" {
			return params, errors.New("currency cannot be empty")
		}
		params.Currency = &trimmed
	}
	if req.Published != nil {
		params.Published = req.Published
	}
	if req.Metadata != nil {
		params.Metadata = *req.Metadata
		params.UpdateMeta = true
	}

	return params, nil
}
