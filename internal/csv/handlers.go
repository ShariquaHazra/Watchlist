package csv

import (
	"net/http"
	"watchlist-backend/pkg/models"

	"github.com/gin-gonic/gin"
)

type Handler struct {
	repo   *Repository
	csvURL string
}

func NewHandler(repo *Repository, csvURL string) *Handler {
	return &Handler{repo: repo, csvURL: csvURL}
}

// POST /api/stocks/import
func (h *Handler) ImportCSV(c *gin.Context) {

	// 🚀 return immediately
	go func() {
		stocks, err := ParseCSV(h.csvURL)
		if err != nil {
			return
		}

		inserted := 0
		failed := 0

		for _, stock := range stocks {
			if err := h.repo.UpsertStock(&stock); err != nil {
				failed++
				continue
			}
			inserted++
		}

		// log result in server logs
		log.Printf("CSV done: inserted=%d failed=%d", inserted, failed)
	}()

	// 🚀 instant response (IMPORTANT)
	c.JSON(http.StatusOK, models.Response{
		Success: true,
		Message: "CSV import started in background",
		Data: gin.H{
			"status": "processing",
		},
	})
}
