package realtime

import (
	"encoding/json"
	"log"
	"sync"
)

// PriceUpdate = ek stock ka live price
type PriceUpdate struct {
	Symbol    string  `json:"symbol"`
	LTP       float64 `json:"ltp"`
	Open      float64 `json:"open"`
	High      float64 `json:"high"`
	Low       float64 `json:"low"`
	Close     float64 `json:"close"`
	Change    float64 `json:"change"`
	ChangePct float64 `json:"change_pct"`
}

// WatchlistUpdate = user ki watchlist stocks
type WatchlistUpdate struct {
	UserID int           `json:"user_id"`
	Stocks []PriceUpdate `json:"stocks"`
}

// Hub = saare connected clients manage karta hai
type Hub struct {
	// Connected clients → userID se map
	clients map[int]*Client

	// Broadcast channel
	broadcast chan WatchlistUpdate

	// Register/Unregister channels
	register   chan *Client
	unregister chan *Client

	mu sync.RWMutex
}

func NewHub() *Hub {
	return &Hub{
		clients:    make(map[int]*Client),
		broadcast:  make(chan WatchlistUpdate, 256),
		register:   make(chan *Client),
		unregister: make(chan *Client),
	}
}

func (h *Hub) Run() {
	for {
		select {
		case client := <-h.register:
			h.mu.Lock()
			h.clients[client.userID] = client
			h.mu.Unlock()
			log.Printf("Client connected: userID=%d (total: %d)", client.userID, len(h.clients))

		case client := <-h.unregister:
			h.mu.Lock()
			if _, ok := h.clients[client.userID]; ok {
				delete(h.clients, client.userID)
				close(client.send)
			}
			h.mu.Unlock()
			log.Printf("Client disconnected: userID=%d (total: %d)", client.userID, len(h.clients))

		case update := <-h.broadcast:
			// Sirf us user ko bhejo jiska userID match kare
			h.mu.RLock()
			client, ok := h.clients[update.UserID]
			h.mu.RUnlock()

			if ok {
				msg, err := json.Marshal(update)
				if err != nil {
					continue
				}
				select {
				case client.send <- msg:
				default:
					h.mu.Lock()
					close(client.send)
					delete(h.clients, update.UserID)
					h.mu.Unlock()
				}
			}
		}
	}
}

// BroadcastToUser — specific user ko price update bhejo
func (h *Hub) BroadcastToUser(update WatchlistUpdate) {
	h.broadcast <- update
}

// GetConnectedUsers — kaun kaun connected hai
func (h *Hub) GetConnectedUsers() []int {
	h.mu.RLock()
	defer h.mu.RUnlock()

	users := make([]int, 0, len(h.clients))
	for userID := range h.clients {
		users = append(users, userID)
	}
	return users
}
