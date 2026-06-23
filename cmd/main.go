package main

import (
	"encoding/json"
	"log"
	"net/http"
	"os"
	"time"

	"github.com/gorilla/mux"

	"watchlist-backend/config"
	"watchlist-backend/internal/auth"
	csvhandler "watchlist-backend/internal/csv"
	"watchlist-backend/internal/db"
	"watchlist-backend/internal/middleware"
	"watchlist-backend/internal/realtime"
	"watchlist-backend/internal/search"
	"watchlist-backend/internal/stock"
	"watchlist-backend/internal/watchlist"
)

func main() {
	cfg := config.Load()

	database := db.Connect(cfg.DatabaseURL)
	defer database.Close()

	// ── Auth ──────────────────────────────────
	authRepo := auth.NewRepository(database)
	authService := auth.NewService(authRepo, cfg.JWTSecret)
	authHandler := auth.NewHandler(authService)

	// ── Watchlist ─────────────────────────────
	watchlistRepo := watchlist.NewRepository(database)
	watchlistService := watchlist.NewService(watchlistRepo)
	watchlistHandler := watchlist.NewHandler(watchlistService)

	// ── Stock ─────────────────────────────────
	stockRepo := stock.NewRepository(database)
	stockService := stock.NewService(stockRepo)
	stockHandler := stock.NewHandler(stockService)

	// ── CSV ───────────────────────────────────
	csvRepo := csvhandler.NewRepository(database)
	csvHandler := csvhandler.NewHandler(csvRepo, cfg.CSVURL)

	// ── Search ────────────────────────────────
	searchRepo := search.NewRepository(database)
	searchService := search.NewService(searchRepo)
	searchHandler := search.NewHandler(searchService)

	// ── Realtime WebSocket ─────────────────────
	hub := realtime.NewHub()
	wsHandler := realtime.NewHandler(hub, database, cfg.JWTSecret)
	priceBroadcaster := realtime.NewPriceBroadcaster(hub, database)

	// ── Background Tasks ──────────────────────
	go hub.Run()
	go priceBroadcaster.Start()

	// ── CSV Auto Load ─────────────────────────
	go func() {
		log.Println("Loading CSV data from URL...")
		stocks, err := csvhandler.ParseCSV(cfg.CSVURL)
		if err != nil {
			log.Printf("CSV load error: %v", err)
			return
		}
		inserted := 0
		for _, s := range stocks {
			if err := csvRepo.UpsertStock(&s); err != nil {
				continue
			}
			inserted++
		}
		log.Printf("CSV loaded: %d stocks inserted/updated", inserted)
	}()

	// ── Router ────────────────────────────────
	r := mux.NewRouter()

	// CORS Middleware
	r.Use(middleware.CORSMiddleware)

	// ── WebSocket — SABSE PEHLE ───────────────
	r.HandleFunc("/ws", wsHandler.ServeWS)

	// OPTIONS preflight
	r.PathPrefix("/").HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNoContent)
	}).Methods("OPTIONS")

	api := r.PathPrefix("/api").Subrouter()

	// ── Health Check ──────────────────────────
	api.HandleFunc("/health", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		if err := database.Ping(); err != nil {
			w.WriteHeader(http.StatusInternalServerError)
			json.NewEncoder(w).Encode(map[string]interface{}{
				"status": "error",
				"db":     "disconnected",
			})
			return
		}
		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(map[string]interface{}{
			"status":    "ok",
			"db":        "connected",
			"timestamp": time.Now(),
		})
	}).Methods("GET")

	// ── Public Routes ─────────────────────────
	api.HandleFunc("/auth/register", authHandler.Register).Methods("POST")
	api.HandleFunc("/auth/login", authHandler.Login).Methods("POST")
	api.HandleFunc("/stocks/import", csvHandler.ImportCSV).Methods("POST")
	api.HandleFunc("/search/stocks", searchHandler.SearchStocks).Methods("GET")

	// Stock Routes
	stockHandler.RegisterRoutes(api)

	// ── Protected Routes ──────────────────────
	protected := api.PathPrefix("/").Subrouter()
	protected.Use(middleware.AuthMiddleware(cfg.JWTSecret))

	protected.HandleFunc("/watchlists", watchlistHandler.Create).Methods("POST")
	protected.HandleFunc("/watchlists", watchlistHandler.GetAll).Methods("GET")
	protected.HandleFunc("/watchlists/{id}", watchlistHandler.Delete).Methods("DELETE")
	protected.HandleFunc("/watchlists/{id}/stocks", watchlistHandler.GetStocks).Methods("GET")
	protected.HandleFunc("/watchlists/{id}/stocks", watchlistHandler.AddStock).Methods("POST")
	protected.HandleFunc("/watchlists/{id}/stocks/{stockId}", watchlistHandler.RemoveStock).Methods("DELETE")

	// ── Server Start ──────────────────────────
	port := os.Getenv("PORT")
	if port == "" {
		port = cfg.ServerPort
	}
	if port == "" {
		port = "8080"
	}
	log.Printf("Server running on port %s", port)
	log.Printf("WebSocket: ws://localhost:%s/ws?token=JWT_TOKEN", port)
	http.ListenAndServe(":"+port, r)
}
