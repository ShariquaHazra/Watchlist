package models

import "time"

type User struct {
	ID          int       `json:"id"`
	Name        string    `json:"name"`
	Email       string    `json:"email"`
	IsBlocked   bool      `json:"is_blocked,omitempty"`
	CreatedAt   time.Time `json:"created_at"`
}

type Stock struct {
	ID                    int       `json:"id"`
	ExchangeInstrumentID  string    `json:"exchange_instrument_id"`
	Segment               string    `json:"segment"`
	InstrumentType        string    `json:"instrument_type"`
	Symbol                string    `json:"symbol"`
	DisplayName           string    `json:"display_name"`
	CompanyName           string    `json:"company_name"`
	ISIN                  string    `json:"isin"`
	Series                string    `json:"series"`
	Exchange              string    `json:"exchange"`
	ContractExpiration    string    `json:"contract_expiration"`
	Strike                float64   `json:"strike"`
	OptionType            string    `json:"option_type"`
	UnderlyingSymbolID    string    `json:"underlying_symbol_id"`
	UnderlyingSymbol      string    `json:"underlying_symbol"`
	LotSize               int       `json:"lot_size"`
	TickSize              float64   `json:"tick_size"`
	UpperCircuit          float64   `json:"upper_circuit"`
	LowerCircuit          float64   `json:"lower_circuit"`
	FreezeQty             int       `json:"freeze_qty"`
	Description           string    `json:"description"`
	LTP                   float64   `json:"ltp"`
	Open                  float64   `json:"open"`
	High                  float64   `json:"high"`
	Low                   float64   `json:"low"`
	Close                 float64   `json:"close"`
	Vol                   int64     `json:"vol"`
	OI                    int64     `json:"oi"`
	Bid                   float64   `json:"bid"`
	Ask                   float64   `json:"ask"`
	BidQty                int       `json:"bid_qty"`
	AskQty                int       `json:"ask_qty"`
	CautionaryMessageInfo string    `json:"cautionary_message_info"`
	LastUpdated           time.Time `json:"last_updated"`
}


// ── Session Control (1 mobile + 1 desktop per user) ───
const (
	DeviceMobile  = "mobile"
	DeviceDesktop = "desktop"
)
 
type UserSession struct {
	ID         int       `json:"id"`
	UserID     int       `json:"user_id"`
	DeviceType string    `json:"device_type"`
	SessionID  string    `json:"session_id"`
	DeviceInfo string    `json:"device_info,omitempty"`
	CreatedAt  time.Time `json:"created_at"`
	LastSeenAt time.Time `json:"last_seen_at"`
}

// SessionInfo — client-facing version, session_id (secret) exposed nahi karta
type SessionInfo struct {
	DeviceType string    `json:"device_type"`
	DeviceInfo string    `json:"device_info,omitempty"`
	CreatedAt  time.Time `json:"created_at"`
	LastSeenAt time.Time `json:"last_seen_at"`
}
 

type Watchlist struct {
	ID         int       `json:"id"`
	UserID     int       `json:"user_id"`
	Name       string    `json:"name"`
	CreatedAt  time.Time `json:"created_at"`
	StockCount int       `json:"stock_count"`
}

type WatchlistItem struct {
	ID          int       `json:"id"`
	WatchlistID int       `json:"watchlist_id"`
	StockID     int       `json:"stock_id"`
	AddedAt     time.Time `json:"added_at"`
	Stock       *Stock    `json:"stock,omitempty"`
}

// ── Request DTOs with Validations ─────────────────────
type RegisterRequest struct {
	Name     string `json:"name"     validate:"required,min=2,max=50,no_only_spaces,valid_name"`
	Email    string `json:"email"    validate:"required,email"`
	Password string `json:"password" validate:"required,min=8,max=100,strong_password"`
	// Optional — agar client explicitly bhejna chahe. Warna User-Agent se auto-detect hoga.
	DeviceType string `json:"device_type" validate:"omitempty,oneof=mobile desktop"`
}
 
type LoginRequest struct {
	Email    string `json:"email"    validate:"required,email"`
	Password string `json:"password" validate:"required"`
	// Optional — agar client explicitly bhejna chahe. Warna User-Agent se auto-detect hoga.
	DeviceType string `json:"device_type" validate:"omitempty,oneof=mobile desktop"`
}

type CreateWatchlistRequest struct {
	Name string `json:"name" validate:"required,min=1,max=100,no_only_spaces,valid_name"`
}

type AddStockRequest struct {
	StockID int `json:"stock_id" validate:"required,min=1"`
}

type AuthResponse struct {
	Token string `json:"token"`
	User  User   `json:"user"`
}

// ── Standard Response ─────────────────────────────────
type Response struct {
	Success bool        `json:"success"`
	Message string      `json:"message"`
	Data    interface{} `json:"data,omitempty"`
}