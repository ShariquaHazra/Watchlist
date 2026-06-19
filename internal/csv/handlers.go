package csv

import (
	"encoding/json"
	"log"
	"net/http"
	"watchlist-backend/pkg/models"
)

type Handler struct {
	repo   *Repository
	csvURL string
}

func NewHandler(repo *Repository, csvURL string) *Handler {
	return &Handler{repo: repo, csvURL: csvURL}
}

// POST /api/stocks/import
func (h *Handler) ImportCSV(w http.ResponseWriter, r *http.Request) {
	stocks, err := ParseCSV(h.csvURL)
	if err != nil {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusInternalServerError)
		json.NewEncoder(w).Encode(models.Response{
			Success: false,
			Message: "failed to parse CSV: " + err.Error(),
		})
		return
	}

	inserted := 0
	failed := 0
	for _, stock := range stocks {
		if err := h.repo.UpsertStock(&stock); err != nil {
			log.Printf("Upsert error: %v", err)
			failed++
			continue
		}
		inserted++
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(models.Response{
		Success: true,
		Message: "CSV imported successfully",
		Data: map[string]int{
			"total_processed":  len(stocks),
			"inserted_updated": inserted,
			"failed":           failed,
		},
	})
}
