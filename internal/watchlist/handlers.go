package watchlist

import (
	"encoding/json"
	"net/http"
	"strconv"
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

// POST /api/watchlists
func (h *Handler) Create(w http.ResponseWriter, r *http.Request) {
	userID := r.Context().Value("user_id").(int)

	var req models.CreateWatchlistRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeJSON(w, http.StatusBadRequest, models.Response{
			Success: false,
			Message: "invalid request body",
		})
		return
	}

	//yeh json body validation ke liye hai, agar koi field missing ya invalid ho toh error return karega
	if errs := validator.Validate(req); len(errs) > 0 {
		writeJSON(w, http.StatusBadRequest, models.Response{
			Success: false,
			Message: "validation failed",
			Data:    errs,
		})
		return
	}

	wl, err := h.service.CreateWatchlist(userID, req.Name)
	if err != nil {
		if err.Error() == "watchlist with this name already exists" {
			writeJSON(w, http.StatusInternalServerError, models.Response{
				Success: false,
				Message: err.Error(),
			})
			return
		}
		writeJSON(w, http.StatusInternalServerError, models.Response{
			Success: false,
			Message: err.Error(), // ← "failed to create watchlist" ki jagah actual error
		})
		return
	}

	writeJSON(w, http.StatusCreated, models.Response{
		Success: true,
		Message: "watchlist created successfully",
		Data:    wl,
	})
}

// GET /api/watchlists
func (h *Handler) GetAll(w http.ResponseWriter, r *http.Request) {
	userID := r.Context().Value("user_id").(int)

	watchlists, err := h.service.GetWatchlists(userID)
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, models.Response{
			Success: false,
			Message: "failed to fetch watchlists",
		})
		return
	}

	if watchlists == nil {
		watchlists = []models.Watchlist{}
	}

	writeJSON(w, http.StatusOK, models.Response{
		Success: true,
		Message: "watchlists fetched successfully",
		Data:    watchlists,
	})
}

// DELETE /api/watchlists/{id}
func (h *Handler) Delete(w http.ResponseWriter, r *http.Request) {
	userID := r.Context().Value("user_id").(int)
	vars := mux.Vars(r)

	watchlistID, err := strconv.Atoi(vars["id"])
	if err != nil {
		writeJSON(w, http.StatusBadRequest, models.Response{
			Success: false,
			Message: "invalid watchlist id",
		})
		return
	}

	if err := h.service.DeleteWatchlist(userID, watchlistID); err != nil {
		if err.Error() == "unauthorized" {
			writeJSON(w, http.StatusForbidden, models.Response{
				Success: false,
				Message: "unauthorized",
			})
			return
		}
		if err.Error() == "watchlist not found" {
			writeJSON(w, http.StatusNotFound, models.Response{
				Success: false,
				Message: "watchlist not found",
			})
			return
		}
		writeJSON(w, http.StatusInternalServerError, models.Response{
			Success: false,
			Message: err.Error(),
		})
		return
	}

	writeJSON(w, http.StatusOK, models.Response{
		Success: true,
		Message: "watchlist deleted successfully",
	})
}

// GET /api/watchlists/{id}/stocks
func (h *Handler) GetStocks(w http.ResponseWriter, r *http.Request) {
	userID := r.Context().Value("user_id").(int)
	vars := mux.Vars(r)

	watchlistID, err := strconv.Atoi(vars["id"])
	if err != nil {
		writeJSON(w, http.StatusBadRequest, models.Response{
			Success: false,
			Message: "invalid watchlist id",
		})
		return
	}

	items, err := h.service.GetStocks(userID, watchlistID)
	if err != nil {
		if err.Error() == "unauthorized" {
			writeJSON(w, http.StatusForbidden, models.Response{
				Success: false,
				Message: "unauthorized",
			})
			return
		}
		if err.Error() == "watchlist not found" {
			writeJSON(w, http.StatusNotFound, models.Response{
				Success: false,
				Message: "watchlist not found",
			})
			return
		}
		writeJSON(w, http.StatusInternalServerError, models.Response{
			Success: false,
			Message: err.Error(),
		})
		return
	}

	if items == nil {
		items = []models.WatchlistItem{}
	}

	writeJSON(w, http.StatusOK, models.Response{
		Success: true,
		Message: "stocks fetched successfully",
		Data:    items,
	})
}

// POST /api/watchlists/{id}/stocks
func (h *Handler) AddStock(w http.ResponseWriter, r *http.Request) {
	userID := r.Context().Value("user_id").(int)
	vars := mux.Vars(r)

	watchlistID, err := strconv.Atoi(vars["id"])
	if err != nil {
		writeJSON(w, http.StatusBadRequest, models.Response{
			Success: false,
			Message: "invalid watchlist id",
		})
		return
	}

	var req models.AddStockRequest
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

	if err := h.service.AddStock(userID, watchlistID, req.StockID); err != nil {
		if err.Error() == "unauthorized" {
			writeJSON(w, http.StatusForbidden, models.Response{
				Success: false,
				Message: "unauthorized",
			})
			return
		}
		if err.Error() == "watchlist not found" {
			writeJSON(w, http.StatusNotFound, models.Response{
				Success: false,
				Message: "watchlist not found",
			})
			return
		}
		if err.Error() == "stock already in watchlist" {
			writeJSON(w, http.StatusBadRequest, models.Response{
				Success: false,
				Message: "stock already in watchlist",
			})
			return
		}
		writeJSON(w, http.StatusInternalServerError, models.Response{
			Success: false,
			Message: err.Error(),
		})
		return
	}

	writeJSON(w, http.StatusCreated, models.Response{
		Success: true,
		Message: "stock added to watchlist successfully",
	})
}

// DELETE /api/watchlists/{id}/stocks/{stockId}
func (h *Handler) RemoveStock(w http.ResponseWriter, r *http.Request) {
	userID := r.Context().Value("user_id").(int)
	vars := mux.Vars(r)

	watchlistID, err := strconv.Atoi(vars["id"])
	if err != nil {
		writeJSON(w, http.StatusBadRequest, models.Response{
			Success: false,
			Message: "invalid watchlist id",
		})
		return
	}

	stockID, err := strconv.Atoi(vars["stockId"])
	if err != nil {
		writeJSON(w, http.StatusBadRequest, models.Response{
			Success: false,
			Message: "invalid stock id",
		})
		return
	}

	if err := h.service.RemoveStock(userID, watchlistID, stockID); err != nil {
		if err.Error() == "unauthorized" {
			writeJSON(w, http.StatusForbidden, models.Response{
				Success: false,
				Message: "unauthorized",
			})
			return
		}
		if err.Error() == "watchlist not found" {
			writeJSON(w, http.StatusNotFound, models.Response{
				Success: false,
				Message: "watchlist not found",
			})
			return
		}
		writeJSON(w, http.StatusInternalServerError, models.Response{
			Success: false,
			Message: err.Error(),
		})
		return
	}

	writeJSON(w, http.StatusOK, models.Response{
		Success: true,
		Message: "stock removed successfully",
	})
}
