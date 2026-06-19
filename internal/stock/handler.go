package stock

import (
	"encoding/json"
	"net/http"

	"watchlist-backend/pkg/models"

	"github.com/gorilla/mux"
)

type StockResponse struct {
	ID          int     `json:"id"`
	Symbol      string  `json:"symbol"`
	CompanyName string  `json:"company_name"`
	Exchange    string  `json:"exchange"`
	LTP         float64 `json:"ltp"`
	LastUpdated string  `json:"last_updated"`
}

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

func toStockResponse(s models.Stock) StockResponse {
	return StockResponse{
		ID:          s.ID,
		Symbol:      s.Symbol,
		CompanyName: s.CompanyName,
		Exchange:    s.Exchange,
		LTP:         s.LTP,
		LastUpdated: s.LastUpdated.Format("2006-01-02 15:04:05"),
	}
}

func (h *Handler) RegisterRoutes(r *mux.Router) {
	r.HandleFunc("/stocks", h.CreateStock).Methods("POST")
	r.HandleFunc("/stocks", h.GetAllStocks).Methods("GET")
	r.HandleFunc("/stocks/{id}", h.GetStockByID).Methods("GET")
	r.HandleFunc("/stocks/{id}", h.UpdateStock).Methods("PUT")
	r.HandleFunc("/stocks/{id}", h.DeleteStock).Methods("DELETE")
}

func (h *Handler) CreateStock(w http.ResponseWriter, r *http.Request) {
	var stock models.Stock
	if err := json.NewDecoder(r.Body).Decode(&stock); err != nil {
		writeJSON(w, http.StatusBadRequest, models.Response{
			Success: false,
			Message: err.Error(),
		})
		return
	}

	if err := h.service.CreateStock(&stock); err != nil {
		writeJSON(w, http.StatusBadRequest, models.Response{
			Success: false,
			Message: err.Error(),
		})
		return
	}

	writeJSON(w, http.StatusCreated, models.Response{
		Success: true,
		Message: "stock created successfully",
		Data:    toStockResponse(stock),
	})
}

func (h *Handler) GetAllStocks(w http.ResponseWriter, r *http.Request) {
	stocks, err := h.service.GetAllStocks()
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, models.Response{
			Success: false,
			Message: err.Error(),
		})
		return
	}

	var response []StockResponse
	for _, s := range stocks {
		response = append(response, toStockResponse(s))
	}

	writeJSON(w, http.StatusOK, models.Response{
		Success: true,
		Message: "stocks fetched successfully",
		Data:    response,
	})
}

func (h *Handler) GetStockByID(w http.ResponseWriter, r *http.Request) {
	id := mux.Vars(r)["id"]

	stock, err := h.service.GetStockByID(id)
	if err != nil {
		writeJSON(w, http.StatusNotFound, models.Response{
			Success: false,
			Message: "stock not found",
		})
		return
	}

	writeJSON(w, http.StatusOK, models.Response{
		Success: true,
		Message: "stock fetched successfully",
		Data:    toStockResponse(*stock),
	})
}

func (h *Handler) UpdateStock(w http.ResponseWriter, r *http.Request) {
	id := mux.Vars(r)["id"]

	var stock models.Stock
	if err := json.NewDecoder(r.Body).Decode(&stock); err != nil {
		writeJSON(w, http.StatusBadRequest, models.Response{
			Success: false,
			Message: err.Error(),
		})
		return
	}

	if err := h.service.UpdateStock(id, &stock); err != nil {
		writeJSON(w, http.StatusBadRequest, models.Response{
			Success: false,
			Message: err.Error(),
		})
		return
	}

	writeJSON(w, http.StatusOK, models.Response{
		Success: true,
		Message: "stock updated successfully",
		Data:    toStockResponse(stock),
	})
}

func (h *Handler) DeleteStock(w http.ResponseWriter, r *http.Request) {
	id := mux.Vars(r)["id"]

	if err := h.service.DeleteStock(id); err != nil {
		writeJSON(w, http.StatusInternalServerError, models.Response{
			Success: false,
			Message: err.Error(),
		})
		return
	}

	writeJSON(w, http.StatusOK, models.Response{
		Success: true,
		Message: "stock deleted successfully",
	})
}
