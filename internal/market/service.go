/*
This package implements a market data service that generates random ticks for subscribed instruments and sends them to connected clients via WebSocket.
It also provides HTTP endpoints for subscribing/unsubscribing to instruments, getting the status of the service, and fetching quotes for specific instruments.
*/
package market

import (
	"encoding/json"
	"log"
	"net/http"

	"watchlist-backend/pkg/models"
)
// type InstrumentMap struct {
// 	Symbol string
// }
type Service struct {
	hub *Hub
	client *Client
	jwtSecret string
}

func NewService(secret string) *Service {//called from main.go to create market service

	h := newHub()
	c := newClient(h)

	return &Service{
		hub: h,
		client: c,
		jwtSecret: secret,

	}
}

func (s *Service) Start() {// Starts the market service by running the hub and starting the client.
	go s.hub.run()//Hub ko new goroutine me run
	s.client.Start()
	log.Println("[market] service started")
}

func (s *Service) Stop() {
	s.client.Stop()
}

// func (s *Service) AddInstrumentMapping(id int64, symbol string) {

// 	s.instruments[id] = InstrumentMap{
// 		Symbol: symbol,
// 	}

// 	log.Println(
// 		"[market] mapped",
// 		id,
// 		symbol,
// 	)
// }

func (s *Service) ServeWS(w http.ResponseWriter, r *http.Request) {
	s.hub.ServeWS(w, r, s.jwtSecret)
}

func (s *Service) HandleSubscribe(w http.ResponseWriter, r *http.Request) {
	var req struct {
		Instruments []struct {
			ExchangeSegment int `json:"exchangeSegment"`
			ExchangeInstrumentID int64 `json:"exchangeInstrumentID"`
		} `json:"instruments"`
		XTSMessageCode int `json:"xtsMessageCode"`
	}

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeErr(w, 400, "invalid request")
		return
	}

	userID, ok := r.Context().Value("user_id").(int)
	if !ok {
		writeErr(w, 401, "unauthorized")
		return
	}

	ins := make([]instrument, len(req.Instruments))

	for i, v := range req.Instruments {
		ins[i] = instrument{
			v.ExchangeSegment,
			v.ExchangeInstrumentID,
		}
	}

	s.client.Subscribe(userID, ins)

	writeOK(w, map[string]interface{}{
		"user_id": userID,
		"subscribed": len(ins),
	})
}

func (s *Service) HandleUnsubscribe(w http.ResponseWriter, r *http.Request) {
	var req struct {
		Instruments []struct {
			ExchangeSegment int `json:"exchangeSegment"`
			ExchangeInstrumentID int64 `json:"exchangeInstrumentID"`
		} `json:"instruments"`
	}

	json.NewDecoder(r.Body).Decode(&req)

	userID, ok := r.Context().Value("user_id").(int)

	if !ok {
		writeErr(w, 401, "unauthorized")
		return
	}

	ins := make([]instrument, len(req.Instruments))

	for i, v := range req.Instruments {
		ins[i] = instrument{
			v.ExchangeSegment,
			v.ExchangeInstrumentID,
		}
	}

	s.client.Unsubscribe(userID, ins)

	writeOK(w, map[string]interface{}{
		"removed": len(ins),
	})
}

func (s *Service) HandleStatus(w http.ResponseWriter, r *http.Request) {
	writeOK(w, map[string]interface{}{
		"frontendClients": s.hub.count(),
		"activeSubscriptions": s.client.ActiveCount(),
	})
}

func (s *Service) HandleQuotes(w http.ResponseWriter, r *http.Request) {

	var req struct {
		Instruments []struct {
			ExchangeSegment int `json:"exchangeSegment"`
			ExchangeInstrumentID int64 `json:"exchangeInstrumentID"`
		} `json:"instruments"`
	}

	json.NewDecoder(r.Body).Decode(&req)

	result := []Tick{}

	for _, v := range req.Instruments {

		tick := randomTick(
			instrument{
				v.ExchangeSegment,
				v.ExchangeInstrumentID,
			},
			nil,
		)

		result = append(result, *tick)
	}

	writeOK(w, result)
}

func (s *Service) HandleWatchlistPrices(w http.ResponseWriter, r *http.Request) {
	var req struct {
		Stocks []struct {
			StockID int `json:"stock_id"`
			Symbol string `json:"symbol"`
			ExchangeInstrumentID string `json:"exchange_instrument_id"`
			ExchangeSegment int `json:"exchange_segment"`
		} `json:"stocks"`
	}

	json.NewDecoder(r.Body).Decode(&req)

	response := []interface{}{}

	for _, st := range req.Stocks {
		tick := randomTick(
			instrument{
				st.ExchangeSegment,
				0,
			},
			nil,
		)

		response = append(response, map[string]interface{}{
			"stock_id": st.StockID,
			"symbol": st.Symbol,
			"ltp": tick.LTP,
			"change": tick.Change,
			"percentChange": tick.PercentChange,
		})
	}

	writeOK(w, response)
}

func writeOK(w http.ResponseWriter, data interface{}) {
	w.Header().Set("Content-Type", "application/json")

	json.NewEncoder(w).Encode(
		models.Response{
			Success: true,
			Data: data,
		},
	)
}

func writeErr(w http.ResponseWriter, status int, msg string) {
	w.WriteHeader(status)

	json.NewEncoder(w).Encode(
		models.Response{
			Success: false,
			Message: msg,
		},
	)
}