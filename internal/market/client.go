package market

import (
	"encoding/json"
	"log"
	"math"
	"math/rand"
	"sync"
	"time"
)

type instrument struct {
	ExchangeSegment int
	ExchangeInstrumentID int64
}

type Tick struct {
	ExchangeSegment int `json:"exchangeSegment"`
	ExchangeInstrumentID int64 `json:"exchangeInstrumentID"`
	LTP float64 `json:"ltp"`
	Open float64 `json:"open"`
	High float64 `json:"high"`
	Low float64 `json:"low"`
	Close float64 `json:"close"`
	Change float64 `json:"change"`
	PercentChange float64 `json:"percentChange"`
	TotalTradedQty int64 `json:"totalTradedQty"`
	LastUpdateTime string `json:"lastUpdateTime"`
}

type tickEnvelope struct {
	Type string `json:"type"`
	Data Tick `json:"data"`
}

type Client struct {
	mu sync.Mutex
	subs map[int]map[instrument]*Tick
	stop chan struct{}
	hub *Hub
}

func newClient(h *Hub) *Client {
	return &Client{
		subs: make(map[int]map[instrument]*Tick),
		stop: make(chan struct{}),
		hub: h,
	}
}

func (c *Client) Start() {
	go c.tickLoop()
	log.Println("[client] tick generator started")
}

func (c *Client) Stop() {
	close(c.stop)
}

func (c *Client) Subscribe(userID int, instruments []instrument) {
	c.mu.Lock()
	defer c.mu.Unlock()

	if c.subs[userID] == nil {
		c.subs[userID] = make(map[instrument]*Tick)
	}

	for _, ins := range instruments {
		c.subs[userID][ins] = nil
	}
}

func (c *Client) Unsubscribe(userID int, instruments []instrument) {
	c.mu.Lock()
	defer c.mu.Unlock()

	userStocks, exists := c.subs[userID]

	if !exists {
		return
	}

	for _, ins := range instruments {
		delete(userStocks, ins)
	}
}

func (c *Client) ActiveCount() int {
	c.mu.Lock()
	defer c.mu.Unlock()

	count := 0

	for _, stocks := range c.subs {
		count += len(stocks)
	}

	return count
}

func (c *Client) tickLoop() {
	ticker := time.NewTicker(1 * time.Second)
	defer ticker.Stop()

	for {
		select {

		case <-c.stop:
			return

		case <-ticker.C:

			c.mu.Lock()

			for userID, stocks := range c.subs {

				for ins, last := range stocks {

					tick := randomTick(
						ins,
						last,
					)

					c.subs[userID][ins] = tick

					data, _ := json.Marshal(
						tickEnvelope{
							Type: "touchline",
							Data: *tick,
						},
					)

					c.hub.sendToUser(userID, data)
				}
			}

			c.mu.Unlock()
		}
	}
}

func randomTick(ins instrument, last *Tick) *Tick {
	rng := rand.New(
		rand.NewSource(time.Now().UnixNano()),
	)

	var price float64
	var open, high, low float64

	if last == nil {
		price = 100 + rng.Float64()*4900
		open = price
		high = price
		low = price
	} else {
		change := last.LTP * (rng.Float64()*0.01 - 0.005)
		price = last.LTP + change
		open = last.Open
		high = math.Max(last.High, price)
		low = math.Min(last.Low, price)
	}

	price = round2(price)

	change := round2(price - open)

	return &Tick{
		ExchangeSegment: ins.ExchangeSegment,
		ExchangeInstrumentID: ins.ExchangeInstrumentID,
		LTP: price,
		Open: round2(open),
		High: round2(high),
		Low: round2(low),
		Close: round2(open),
		Change: change,
		PercentChange: round2((change/open)*100),
		TotalTradedQty: int64(rng.Intn(10000000)),
		LastUpdateTime: time.Now().Format("2006-01-02 15:04:05"),
	}
}

func round2(v float64) float64 {
	return math.Round(v*100) / 100
}