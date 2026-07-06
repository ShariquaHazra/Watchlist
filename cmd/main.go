package main

import (
	"bufio"
	"encoding/json" 
	"log"
	"net"
	"net/http"
	"os"
	"time"

	"github.com/gorilla/mux" 
	"watchlist-backend/config"
	"watchlist-backend/internal/auth"
	csvhandler "watchlist-backend/internal/csv"
	"watchlist-backend/internal/db"
	"watchlist-backend/internal/market"
	"watchlist-backend/internal/middleware"
	"watchlist-backend/internal/search"
	"watchlist-backend/internal/stock"
	"watchlist-backend/internal/watchlist"
)

type statusWriter struct {
	http.ResponseWriter
	status int
}

func (sw *statusWriter) WriteHeader(code int) {
	sw.status = code
	sw.ResponseWriter.WriteHeader(code)
}

func (sw *statusWriter) Hijack() (net.Conn, *bufio.ReadWriter, error) {
	h, ok := sw.ResponseWriter.(http.Hijacker)
	if !ok {
		return nil, nil, http.ErrNotSupported
	}
	return h.Hijack()
}

func loggingMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()
		sw := &statusWriter{ResponseWriter: w, status: http.StatusOK}

		next.ServeHTTP(sw, r)

		log.Printf("[MUX] %s | %3d | %13v | %15s | %-7s %q",
			time.Now().Format("2006/01/02 - 15:04:05"),
			sw.status,
			time.Since(start),
			r.RemoteAddr,
			r.Method,
			r.URL.Path,
		)
	})
}

func main() {
	cfg := config.Load()

	dbConn := db.Connect(cfg.DatabaseURL)//Ye database se connection banata hai.
	defer dbConn.Close()

	// ── Auth ─────────────────────────
	authRepo := auth.NewRepository(dbConn)
	authService := auth.NewService(authRepo, cfg.JWTSecret)
	authHandler := auth.NewHandler(authService)

	// go func() {
	// 	ticker := time.NewTicker(1 * time.Hour)
	// 	defer ticker.Stop()
	// 	for range ticker.C {
	// 		deleted, err := authRepo.CleanupExpiredRevokedTokens()
	// 		if err != nil {
	// 			log.Println("revoked_tokens cleanup failed:", err)
	// 			continue
	// 		}
	// 		if deleted > 0 {
	// 			log.Printf("revoked_tokens cleanup: removed %d expired entries\n", deleted)
	// 		}
	// 	}
	// }()

	// ── Watchlist ────────────────────
	watchlistRepo := watchlist.NewRepository(dbConn)
	watchlistService := watchlist.NewService(watchlistRepo)
	watchlistHandler := watchlist.NewHandler(watchlistService)

	// ── Stock ────────────────────────
	stockRepo := stock.NewRepository(dbConn)
	stockService := stock.NewService(stockRepo)
	stockHandler := stock.NewHandler(stockService)

	// ── CSV ──────────────────────────
	csvRepo := csvhandler.NewRepository(dbConn)
	csvHandler := csvhandler.NewHandler(csvRepo, cfg.CSVURL)

	// ── Search ───────────────────────
	searchRepo := search.NewRepository(dbConn)
	searchService := search.NewService(searchRepo)
	searchHandler := search.NewHandler(searchService)

	// ── CSV Loader ───────────────────
	go func() {
		log.Println("Loading CSV data from URL...")

		stocks, err := csvhandler.ParseCSV(cfg.CSVURL)
		if err != nil {
			log.Printf("CSV load error: %v", err)
			return //go routine ko terminate kar deta hai agar error aata hai.
		}

		inserted := 0
		for _, s := range stocks {
			if err := csvRepo.UpsertStock(&s); err == nil {
				inserted++
			}
		}

		log.Printf("CSV loaded: %d stocks inserted/updated", inserted)
	}()

	// ── MARKET ENGINE (IMPORTANT) ────
	marketSvc := market.NewService(cfg.JWTSecret)
	marketSvc.Start()
	defer marketSvc.Stop()

	// ── ROUTER ───────────────────────
	r := mux.NewRouter()
	r.Use(middleware.CORSMiddleware)
	r.Use(loggingMiddleware)

	r.PathPrefix("/").HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNoContent)
	}).Methods("OPTIONS")

	api := r.PathPrefix("/api").Subrouter()

	// ── Public APIs ──────────────────
	api.HandleFunc("/health", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")

		if err := dbConn.Ping(); err != nil {
			w.WriteHeader(http.StatusInternalServerError)
			json.NewEncoder(w).Encode(map[string]string{
				"status": "error",
				"db":     "disconnected",
			})
			return
		}

		json.NewEncoder(w).Encode(map[string]interface{}{
			"status": "ok",
			"db":     "connected",
			"time":   time.Now(),
		})
	}).Methods("GET")

	api.HandleFunc("/auth/register", authHandler.Register).Methods("POST")
	api.HandleFunc("/auth/login", authHandler.Login).Methods("POST")
	api.HandleFunc("/stocks/import", csvHandler.ImportCSV).Methods("POST")
	api.HandleFunc("/search/stocks", searchHandler.SearchStocks).Methods("GET")

	stockHandler.RegisterRoutes(api)// Register stock-related routes

	// ── WebSocket ─────────────────────
	api.HandleFunc("/ws/market", marketSvc.ServeWS).Methods("GET")

	// ── Protected APIs ───────────────
	protected := api.NewRoute().Subrouter()
	protected.Use(middleware.AuthMiddleware(cfg.JWTSecret, dbConn))


	protected.HandleFunc("/auth/logout", authHandler.Logout).Methods("POST")
	protected.HandleFunc("/auth/sessions", authHandler.GetSessions).Methods("GET")
	protected.HandleFunc("/auth/sessions/{device_type}", authHandler.DeleteSessionByType).Methods("DELETE")

	protected.HandleFunc("/watchlists", watchlistHandler.Create).Methods("POST")
	protected.HandleFunc("/watchlists", watchlistHandler.GetAll).Methods("GET")
	protected.HandleFunc("/watchlists/{id}", watchlistHandler.Delete).Methods("DELETE")

	protected.HandleFunc("/watchlists/{id}/stocks", watchlistHandler.GetStocks).Methods("GET")
	protected.HandleFunc("/watchlists/{id}/stocks", watchlistHandler.AddStock).Methods("POST")
	protected.HandleFunc("/watchlists/{id}/stocks/{stockId}", watchlistHandler.RemoveStock).Methods("DELETE")

	// ── MARKET APIs ( SUBSCRIBE) ─
	protected.HandleFunc("/market/subscribe", marketSvc.HandleSubscribe).Methods("POST")
	protected.HandleFunc("/market/unsubscribe", marketSvc.HandleUnsubscribe).Methods("POST")
	protected.HandleFunc("/market/quotes", marketSvc.HandleQuotes).Methods("POST")
	protected.HandleFunc("/market/watchlist-prices", marketSvc.HandleWatchlistPrices).Methods("POST")
	protected.HandleFunc("/market/status", marketSvc.HandleStatus).Methods("GET")

	// ── SERVER START ──────────────────
	port := os.Getenv("PORT")
	if port == "" {
		port = cfg.ServerPort
	}
	if port == "" {
		port = "8080"
	}

	log.Printf("Server running on port %s", port)

	if err := http.ListenAndServe(":"+port, r); err != nil {
		log.Fatal(err)
	}
}