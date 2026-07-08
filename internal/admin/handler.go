package admin

import (
	"encoding/json"
	"net/http"
	"strconv"
	"watchlist-backend/pkg/models"

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

// DELETE /api/admin/users/{userID}/sessions/{deviceType} (admin only)
// Kisi bhi user ka ek specific device session force-revoke — us user ke logout
// kiye bina. Notice: r.Context().Value("user_id") yahan use NAHI hota, kyunki
// wo calling ADMIN ki ID hai — target user URL se aata hai (vars["userID"]).
func (h *Handler) RevokeUserSession(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)

	userID, err := strconv.Atoi(vars["userID"])
	if err != nil {
		writeJSON(w, http.StatusBadRequest, models.Response{
			Success: false,
			Message: "invalid user id",
		})
		return
	}

	deviceType := vars["deviceType"]

	if err := h.service.RevokeUserSession(userID, deviceType); err != nil {
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

// DELETE /api/admin/users/{userID}/sessions (admin only)
// Kisi bhi user ke saare devices (mobile + desktop) se force-logout — "kick out
// everywhere" jaisi situation ke liye (account-compromise, security incident).
func (h *Handler) RevokeAllSessions(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)

	userID, err := strconv.Atoi(vars["userID"])
	if err != nil {
		writeJSON(w, http.StatusBadRequest, models.Response{
			Success: false,
			Message: "invalid user id",
		})
		return
	}

	if err := h.service.RevokeAllUserSessions(userID); err != nil {
		writeJSON(w, http.StatusInternalServerError, models.Response{
			Success: false,
			Message: "failed to revoke sessions",
		})
		return
	}

	writeJSON(w, http.StatusOK, models.Response{
		Success: true,
		Message: "all sessions revoked successfully",
	})
}

func (h *Handler) BlockUser(w http.ResponseWriter, r *http.Request) {

	vars := mux.Vars(r)

	userID, err := strconv.Atoi(vars["userID"])
	if err != nil {
		writeJSON(w, http.StatusBadRequest, models.Response{
			Success: false,
			Message: "invalid user id",
		})
		return
	}

	if err := h.service.BlockUser(userID); err != nil {
		writeJSON(w, http.StatusInternalServerError, models.Response{
			Success: false,
			Message: err.Error(),
		})
		return
	}

	writeJSON(w, http.StatusOK, models.Response{
		Success: true,
		Message: "user blocked successfully",
	})
}

func (h *Handler) UnblockUser(w http.ResponseWriter, r *http.Request) {

	vars := mux.Vars(r)

	userID, err := strconv.Atoi(vars["userID"])
	if err != nil {
		writeJSON(w, http.StatusBadRequest, models.Response{
			Success: false,
			Message: "invalid user id",
		})
		return
	}

	if err := h.service.UnblockUser(userID); err != nil {
		writeJSON(w, http.StatusInternalServerError, models.Response{
			Success: false,
			Message: err.Error(),
		})
		return
	}

	writeJSON(w, http.StatusOK, models.Response{
		Success: true,
		Message: "user unblocked successfully",
	})
}