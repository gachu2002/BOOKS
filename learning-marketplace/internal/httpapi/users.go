package httpapi

import (
	"errors"
	"net/http"
	"strings"

	"github.com/go-chi/chi/v5"

	"learning-marketplace/internal/app"
	"learning-marketplace/internal/store"
)

type createUserRequest struct {
	Email    string `json:"email"`
	FullName string `json:"full_name"`
	Role     string `json:"role"`
}

type updateUserRequest struct {
	Email    *string `json:"email"`
	FullName *string `json:"full_name"`
	Role     *string `json:"role"`
}

func registerUserRoutes(r chi.Router, application *app.App) {
	r.Route("/users", func(r chi.Router) {
		r.Post("/", func(w http.ResponseWriter, r *http.Request) {
			var req createUserRequest
			if err := decodeJSON(r, &req); err != nil {
				writeError(w, http.StatusBadRequest, err.Error())
				return
			}

			if err := validateCreateUser(req); err != nil {
				writeError(w, http.StatusBadRequest, err.Error())
				return
			}

			user, err := application.Store.CreateUser(r.Context(), store.UserCreateParams{
				Email:    strings.TrimSpace(req.Email),
				FullName: strings.TrimSpace(req.FullName),
				Role:     defaultUserRole(req.Role),
			})
			if err != nil {
				writeStoreError(w, err)
				return
			}

			writeJSON(w, http.StatusCreated, user)
		})

		r.Get("/", func(w http.ResponseWriter, r *http.Request) {
			page := parsePagination(r)
			users, err := application.Store.ListUsers(r.Context(), store.Pagination{Limit: page.Limit, Offset: page.Offset})
			if err != nil {
				writeStoreError(w, err)
				return
			}

			writeJSON(w, http.StatusOK, map[string]any{
				"items":  users,
				"limit":  page.Limit,
				"offset": page.Offset,
			})
		})

		r.Get("/{userID}", func(w http.ResponseWriter, r *http.Request) {
			user, err := application.Store.GetUser(r.Context(), pathID(r, "userID"))
			if err != nil {
				writeStoreError(w, err)
				return
			}

			writeJSON(w, http.StatusOK, user)
		})

		r.Patch("/{userID}", func(w http.ResponseWriter, r *http.Request) {
			var req updateUserRequest
			if err := decodeJSON(r, &req); err != nil {
				writeError(w, http.StatusBadRequest, err.Error())
				return
			}

			params, err := validateUpdateUser(req)
			if err != nil {
				writeError(w, http.StatusBadRequest, err.Error())
				return
			}

			user, err := application.Store.UpdateUser(r.Context(), pathID(r, "userID"), params)
			if err != nil {
				writeStoreError(w, err)
				return
			}

			writeJSON(w, http.StatusOK, user)
		})

		r.Delete("/{userID}", func(w http.ResponseWriter, r *http.Request) {
			if err := application.Store.DeleteUser(r.Context(), pathID(r, "userID")); err != nil {
				writeStoreError(w, err)
				return
			}

			w.WriteHeader(http.StatusNoContent)
		})
	})
}

func validateCreateUser(req createUserRequest) error {
	if err := requireString(req.Email, "email"); err != nil {
		return err
	}
	if err := requireString(req.FullName, "full_name"); err != nil {
		return err
	}

	role := defaultUserRole(req.Role)
	if role != "student" && role != "admin" {
		return errors.New("role must be one of: student, admin")
	}

	return nil
}

func validateUpdateUser(req updateUserRequest) (store.UserUpdateParams, error) {
	params := store.UserUpdateParams{}
	if req.Email != nil {
		trimmed := strings.TrimSpace(*req.Email)
		if trimmed == "" {
			return store.UserUpdateParams{}, errors.New("email cannot be empty")
		}
		params.Email = &trimmed
	}
	if req.FullName != nil {
		trimmed := strings.TrimSpace(*req.FullName)
		if trimmed == "" {
			return store.UserUpdateParams{}, errors.New("full_name cannot be empty")
		}
		params.FullName = &trimmed
	}
	if req.Role != nil {
		trimmed := strings.TrimSpace(*req.Role)
		if trimmed != "student" && trimmed != "admin" {
			return store.UserUpdateParams{}, errors.New("role must be one of: student, admin")
		}
		params.Role = &trimmed
	}

	return params, nil
}

func defaultUserRole(role string) string {
	trimmed := strings.TrimSpace(role)
	if trimmed == "" {
		return "student"
	}

	return trimmed
}
