package httpapi

import (
	"database/sql"
	"errors"
	"fmt"
	"net/http"
	"strings"

	"github.com/jackc/pgx/v5/pgconn"

	"learning-marketplace/internal/store"
)

type errorResponse struct {
	Error string `json:"error"`
}

func writeError(w http.ResponseWriter, status int, message string) {
	writeJSON(w, status, errorResponse{Error: message})
}

func writeStoreError(w http.ResponseWriter, err error) {
	switch {
	case errors.Is(err, store.ErrNotFound):
		writeError(w, http.StatusNotFound, "resource not found")
	default:
		var pgErr *pgconn.PgError
		if errors.As(err, &pgErr) {
			switch pgErr.Code {
			case "23505":
				writeError(w, http.StatusConflict, humanizeConstraint(pgErr.ConstraintName))
				return
			case "23503", "23514":
				writeError(w, http.StatusBadRequest, pgErr.Message)
				return
			}
		}

		if errors.Is(err, sql.ErrNoRows) {
			writeError(w, http.StatusNotFound, "resource not found")
			return
		}

		writeError(w, http.StatusInternalServerError, "internal server error")
	}
}

func humanizeConstraint(name string) string {
	if name == "" {
		return "constraint violation"
	}

	name = strings.ReplaceAll(name, "_", " ")
	return fmt.Sprintf("constraint violation: %s", name)
}
