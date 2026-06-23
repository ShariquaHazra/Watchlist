package realtime

import (
	"database/sql"
	"encoding/json"
	"log"
	"math/rand/v2"
	"net/http"
	"strings"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/gorilla/websocket"
)

var upgrader = websocket.Upgrader{
	ReadBufferSize:  1024,
	WriteBufferSize: 1024,
	CheckOrigin: func(r *http.Request) bool {
		return true // Production mein specific origin check karo
	},
}

type Handler struct {
	hub       *Hub
	db        *sql.DB
	jwtSecret string
}

func NewHandler(hub *Hub, db *sql.DB, jwtSecret string) *Handler {
	return &Handler{hub: hub, db: db, jwtSecret: jwtSecret}
}

// GET /ws?token=JWT_TOKEN
func (h *Handler) ServeWS(w http.ResponseWriter, r *http.Request) {

	// Token query param se lo
	tokenStr := r.URL.Query().Get("token")
	if tokenStr == "" {
		// Header se bhi try karo
		authHeader := r.Header.Get("Authorization")
		if authHeader != "" && strings.HasPrefix(authHeader, "Bearer ") {
			tokenStr = strings.TrimPrefix(authHeader, "Bearer ")
		}
	}

	if tokenStr == "" {
		http.Error(w, "token required", http.StatusUnauthorized)
		return
	}

	// Token verify karo
	token, err := jwt.Parse(tokenStr, func(t *jwt.Token) (interface{}, error) {
		return []byte(h.jwtSecret), nil
	})
	if err != nil || !token.Valid {
		http.Error(w, "invalid token", http.StatusUnauthorized)
		return
	}

	// user_id nikalo
	claims := token.Claims.(jwt.MapClaims)
	userID := int(claims["user_id"].(float64))

	// HTTP → WebSocket upgrade
	conn, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		log.Printf("WebSocket upgrade error: %v", err)
		return
	}

	client := &Client{
		hub:    h.hub,
		conn:   conn,
		send:   make(chan []byte, 256),
		userID: userID,
	}

	h.hub.register <- client

	// Welcome message bhejo
	welcome := map[string]interface{}{
		"type":    "connected",
		"message": "WebSocket connected successfully",
		"user_id": userID,
	}
	welcomeMsg, _ := json.Marshal(welcome)
	client.send <- welcomeMsg

	// Read aur Write alag goroutines mein chalao
	go client.WritePump()
	go client.ReadPump()
}

// PriceBroadcaster = background mein DB se prices fetch karta hai aur broadcast karta hai
type PriceBroadcaster struct {
	hub    *Hub
	db     *sql.DB
	prices map[string]float64
}

func NewPriceBroadcaster(hub *Hub, db *sql.DB) *PriceBroadcaster {
	return &PriceBroadcaster{hub: hub, db: db, prices: make(map[string]float64)}
}

// Start — background mein har 3 second mein prices broadcast karo
func (pb *PriceBroadcaster) Start() {
	log.Println("Price broadcaster started")

	// Pehle DB se actual prices load karo
	pb.loadInitialPrices()

	ticker := time.NewTicker(3 * time.Second)
	defer ticker.Stop()

	for range ticker.C {
		pb.broadcastToAllUsers()
	}
}

func (pb *PriceBroadcaster) loadInitialPrices() {
	rows, err := pb.db.Query(`SELECT symbol, ltp FROM stocks WHERE ltp > 0`)
	if err != nil {
		log.Printf("Error loading initial prices: %v", err)
		return
	}
	defer rows.Close()

	for rows.Next() {
		var symbol string
		var ltp float64
		if err := rows.Scan(&symbol, &ltp); err != nil {
			continue
		}
		pb.prices[symbol] = ltp
	}
	log.Printf("Loaded %d stock prices for simulation", len(pb.prices))
}

// Price simulate karo — +/- 0.5% random change
func (pb *PriceBroadcaster) simulatePrice(symbol string, currentPrice float64) float64 {
	if currentPrice <= 0 {
		return currentPrice
	}

	// +/- 0.5% random change
	changePct := (rand.Float64() - 0.5) * 1.0
	change := currentPrice * changePct / 100
	newPrice := currentPrice + change

	// Negative price nahi honi chahiye
	if newPrice <= 0 {
		newPrice = currentPrice
	}

	// In-memory update karo
	pb.prices[symbol] = newPrice
	return newPrice
}

func (pb *PriceBroadcaster) broadcastToAllUsers() {
	// Saare connected users
	connectedUsers := pb.hub.GetConnectedUsers()
	if len(connectedUsers) == 0 {
		return
	}

	for _, userID := range connectedUsers {
		// Us user ki watchlist stocks fetch karo
		stocks, err := pb.getUserWatchlistPrices(userID)
		if err != nil {
			log.Printf("Error fetching prices for user %d: %v", userID, err)
			continue
		}

		if len(stocks) == 0 {
			continue
		}

		// Broadcast karo
		pb.hub.BroadcastToUser(WatchlistUpdate{
			UserID: userID,
			Stocks: stocks,
		})
	}
}

func (pb *PriceBroadcaster) getUserWatchlistPrices(userID int) ([]PriceUpdate, error) {
	// User ki saari watchlists ke stocks fetch karo
	query := `
		SELECT DISTINCT
			s.symbol,
			s.ltp,
			s.open,
			s.high,
			s.low,
			s.close
		FROM watchlist_items wi
		JOIN watchlists w ON w.id = wi.watchlist_id
		JOIN stocks s ON s.id = wi.stock_id
		WHERE w.user_id = $1
	`

	rows, err := pb.db.Query(query, userID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var stocks []PriceUpdate
	for rows.Next() {
		var symbol string
		var ltp, open, high, low, close float64

		err := rows.Scan(&symbol, &ltp, &open, &high, &low, &close)
		if err != nil {
			continue
		}

		// In-memory price use karo — simulate karo
		currentPrice, exists := pb.prices[symbol]
		if !exists {
			currentPrice = ltp
		}
		simulatedPrice := pb.simulatePrice(symbol, currentPrice)

		change := simulatedPrice - close
		changePct := 0.0
		if close > 0 {
			changePct = (change / close) * 100
		}

		stocks = append(stocks, PriceUpdate{
			Symbol:    symbol,
			LTP:       simulatedPrice,
			Open:      open,
			High:      high,
			Low:       low,
			Close:     close,
			Change:    change,
			ChangePct: changePct,
		})
	}

	return stocks, nil
}
