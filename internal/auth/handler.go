package auth

import (
	"encoding/json"
	"net/http"
	"watchlist-backend/pkg/models"
	"watchlist-backend/pkg/validator"
)

type Handler struct {
	service *Service
}

func NewHandler(service *Service) *Handler {
	return &Handler{service: service}
}

func writeJSON(w http.ResponseWriter, status int, data interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(data)
}

// POST /api/auth/register
func (h *Handler) Register(w http.ResponseWriter, r *http.Request) {
	var req models.RegisterRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeJSON(w, http.StatusBadRequest, models.Response{
			Success: false,
			Message: "invalid request body",
		})
		return
	}

	// Validation
	if errs := validator.Validate(req); len(errs) > 0 {
		writeJSON(w, http.StatusBadRequest, models.Response{
			Success: false,
			Message: "validation failed",
			Data:    errs,
		})
		return
	}

	resp, err := h.service.Register(&req)
	if err != nil {
		writeJSON(w, http.StatusBadRequest, models.Response{
			Success: false,
			Message: err.Error(),
		})
		return
	}

	writeJSON(w, http.StatusCreated, models.Response{
		Success: true,
		Message: "user registered successfully",
		Data:    resp,
	})
}

// POST /api/auth/login
func (h *Handler) Login(w http.ResponseWriter, r *http.Request) {
	var req models.LoginRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeJSON(w, http.StatusBadRequest, models.Response{
			Success: false,
			Message: "invalid request body",
		})
		return
	}

	// Validation
	if errs := validator.Validate(req); len(errs) > 0 {
		writeJSON(w, http.StatusBadRequest, models.Response{
			Success: false,
			Message: "validation failed",
			Data:    errs,
		})
		return
	}

	resp, err := h.service.Login(&req)
	if err != nil {
		writeJSON(w, http.StatusUnauthorized, models.Response{
			Success: false,
			Message: err.Error(),
		})
		return
	}

	writeJSON(w, http.StatusOK, models.Response{
		Success: true,
		Message: "login successful",
		Data:    resp,
	})
}
