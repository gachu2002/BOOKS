package httpapi

import (
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"strconv"
	"strings"

	"github.com/go-chi/chi/v5"
)

func decodeJSON(r *http.Request, dst any) error {
	decoder := json.NewDecoder(r.Body)
	decoder.DisallowUnknownFields()

	if err := decoder.Decode(dst); err != nil {
		return err
	}

	if err := decoder.Decode(new(struct{})); err != nil {
		if errors.Is(err, io.EOF) {
			return nil
		}
		return errors.New("request body must contain a single JSON object")
	}

	return errors.New("request body must contain a single JSON object")
}

func parsePagination(r *http.Request) Pagination {
	limit := 20
	offset := 0

	if raw := r.URL.Query().Get("limit"); raw != "" {
		if parsed, err := strconv.Atoi(raw); err == nil && parsed > 0 && parsed <= 100 {
			limit = parsed
		}
	}

	if raw := r.URL.Query().Get("offset"); raw != "" {
		if parsed, err := strconv.Atoi(raw); err == nil && parsed >= 0 {
			offset = parsed
		}
	}

	return Pagination{Limit: limit, Offset: offset}
}

type Pagination struct {
	Limit  int `json:"limit"`
	Offset int `json:"offset"`
}

func pathID(r *http.Request, key string) string {
	return chi.URLParam(r, key)
}

func requireString(value, field string) error {
	if strings.TrimSpace(value) == "" {
		return errors.New(field + " is required")
	}

	return nil
}
