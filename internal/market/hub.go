package market

import (
	"fmt"
	"log"
	"net/http"
	"sync"

	"github.com/golang-jwt/jwt/v5"
	"github.com/gorilla/websocket"
)

var upgrader = websocket.Upgrader{
	CheckOrigin: func(r *http.Request) bool {
		return true
	},
}

type Hub struct {
	mu sync.RWMutex
	clients map[*wsClient]struct{}
}

func newHub() *Hub {
	return &Hub{
		clients: make(map[*wsClient]struct{}),
	}
}

func (h *Hub) run() {
	// ticks are directly sent using sendToUser()
}

func (h *Hub) register(c *wsClient) {
	h.mu.Lock()
	h.clients[c] = struct{}{}
	h.mu.Unlock()

	log.Printf("[hub] user %d connected", c.userID)
}

func (h *Hub) remove(c *wsClient) {
	h.mu.Lock()
	defer h.mu.Unlock()

	if _, ok := h.clients[c]; ok {
		delete(h.clients, c)
		close(c.send)
		c.conn.Close()
	}
}

func (h *Hub) count() int {
	h.mu.RLock()
	defer h.mu.RUnlock()

	return len(h.clients)
}

func (h *Hub) sendToUser(userID int, msg []byte) {
	h.mu.RLock()
	defer h.mu.RUnlock()

	for c := range h.clients {

		if c.userID == userID {

			select {
			case c.send <- msg:

			default:
				log.Println("client slow")
			}
		}
	}
}

type wsClient struct {
	hub *Hub
	conn *websocket.Conn
	send chan []byte
	userID int
}

func (h *Hub) ServeWS(
	w http.ResponseWriter,
	r *http.Request,
	secret string,
) {

	tokenString := r.URL.Query().Get("token")

	userID, ok := validateToken(tokenString, secret)

	if !ok {
		http.Error(
			w,
			"invalid token",
			401,
		)
		return
	}

	conn, err := upgrader.Upgrade(
		w,
		r,
		nil,
	)

	if err != nil {
		return
	}

	client := &wsClient{
		hub: h,
		conn: conn,
		send: make(chan []byte, 256),
		userID: userID,
	}

	h.register(client)

	go client.write()
	go client.read()
}

func (c *wsClient) write() {

	for msg := range c.send {

		err := c.conn.WriteMessage(
			websocket.TextMessage,
			msg,
		)

		if err != nil {
			return
		}
	}
}

func (c *wsClient) read() {

	defer c.hub.remove(c)

	for {

		_, _, err := c.conn.ReadMessage()

		if err != nil {
			return
		}
	}
}

func validateToken(
	tokenString string,
	secret string,
)(int,bool){

	token, err := jwt.Parse(
		tokenString,

		func(token *jwt.Token)(interface{},error){

			if _,ok := token.Method.(*jwt.SigningMethodHMAC); !ok {

				return nil, fmt.Errorf(
					"invalid signing method",
				)
			}

			return []byte(secret),nil
		},
	)

	if err != nil || !token.Valid {
		return 0,false
	}

	claims,ok := token.Claims.(jwt.MapClaims)

	if !ok {
		return 0,false
	}

	id,ok := claims["user_id"].(float64)

	if !ok {
		return 0,false
	}

	return int(id),true
}