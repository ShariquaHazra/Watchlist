package auth

import (
	"encoding/json"
	"net/http"
	"watchlist-backend/pkg/models"
	"watchlist-backend/pkg/validator"

	"github.com/gorilla/mux"
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

	resp, err := h.service.Register(&req, r.Header.Get("User-Agent"))
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

	resp, err := h.service.Login(&req, r.Header.Get("User-Agent"))
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

// POST /api/auth/logout (protected — AuthMiddleware ke baad chalta hai)
func (h *Handler) Logout(w http.ResponseWriter, r *http.Request) {
	userID, _ := r.Context().Value("user_id").(int)
	deviceType, _ := r.Context().Value("device_type").(string)

	if err := h.service.Logout(userID, deviceType); err != nil {
		writeJSON(w, http.StatusInternalServerError, models.Response{
			Success: false,
			Message: "logout failed",
		})
		return
	}

	writeJSON(w, http.StatusOK, models.Response{
		Success: true,
		Message: "logged out successfully",
	})
}

// GET /api/auth/sessions (protected) — user ke saare active sessions (mobile + desktop)
func (h *Handler) GetSessions(w http.ResponseWriter, r *http.Request) {
	userID, _ := r.Context().Value("user_id").(int)

	sessions, err := h.service.ListSessions(userID)
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, models.Response{
			Success: false,
			Message: "failed to fetch sessions",
		})
		return
	}

	writeJSON(w, http.StatusOK, models.Response{
		Success: true,
		Message: "active sessions fetched",
		Data:    sessions,
	})
}

// DELETE /api/auth/sessions/{device_type} (protected) — remote-logout ek specific
// device_type (mobile/desktop) se, chahe wo request khud us device se aayi ho ya kisi aur se
func (h *Handler) DeleteSessionByType(w http.ResponseWriter, r *http.Request) {
	userID, _ := r.Context().Value("user_id").(int)
	deviceType := mux.Vars(r)["device_type"]

	if err := h.service.RevokeSession(userID, deviceType); err != nil {
		writeJSON(w, http.StatusBadRequest, models.Response{
			Success: false,
			Message: err.Error(),
		})
		return
	}

	writeJSON(w, http.StatusOK, models.Response{
		Success: true,
		Message: deviceType + " session revoked successfully",
	})
}