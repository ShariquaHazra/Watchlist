package watchlist

import (
	"database/sql"
	"errors"
	"watchlist-backend/pkg/models"
)

type Repository struct {
	db *sql.DB
}

func NewRepository(db *sql.DB) *Repository {
	return &Repository{db: db}
}

// Same user ki same name ki watchlist exist karti hai?
func (r *Repository) ExistsByName(userID int, name string) (bool, error) {
	query := `
		SELECT COUNT(*) FROM watchlists
		WHERE user_id = $1
		AND LOWER(TRIM(name)) = LOWER(TRIM($2))
	`
	var count int
	err := r.db.QueryRow(query, userID, name).Scan(&count)
	return count > 0, err
}

func (r *Repository) Create(w *models.Watchlist) error {
	query := `
		INSERT INTO watchlists (user_id, name, created_at)
		VALUES ($1, $2, NOW())
		RETURNING id
	`
	return r.db.QueryRow(query, w.UserID, w.Name).Scan(&w.ID)
}

func (r *Repository) GetAllByUserID(userID int) ([]models.Watchlist, error) {
	query := `
		SELECT w.id, w.user_id, w.name, w.created_at,
		       COUNT(wi.id) as stock_count
		FROM watchlists w
		LEFT JOIN watchlist_items wi ON wi.watchlist_id = w.id
		WHERE w.user_id = $1
		GROUP BY w.id
		ORDER BY w.created_at DESC
	`
	rows, err := r.db.Query(query, userID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var watchlists []models.Watchlist
	for rows.Next() {
		var w models.Watchlist
		if err := rows.Scan(&w.ID, &w.UserID, &w.Name, &w.CreatedAt, &w.StockCount); err != nil {
			return nil, err
		}
		watchlists = append(watchlists, w)
	}
	return watchlists, nil
}

func (r *Repository) GetByID(watchlistID int) (*models.Watchlist, error) {
	query := `SELECT id, user_id, name, created_at FROM watchlists WHERE id = $1`
	w := &models.Watchlist{}
	err := r.db.QueryRow(query, watchlistID).Scan(&w.ID, &w.UserID, &w.Name, &w.CreatedAt)
	if err == sql.ErrNoRows {
		return nil, errors.New("watchlist not found")
	}
	return w, err
}

func (r *Repository) Delete(watchlistID int) error {
	_, err := r.db.Exec(`DELETE FROM watchlists WHERE id = $1`, watchlistID)
	return err
}

func (r *Repository) GetStocks(watchlistID int) ([]models.WatchlistItem, error) {
	query := `
		SELECT
			wi.id, wi.watchlist_id, wi.stock_id, wi.added_at,
			s.id, s.exchange_instrument_id, s.segment, s.instrument_type,
			s.symbol, s.display_name, s.company_name, s.isin, s.series,
			s.exchange, s.contract_expiration, s.strike, s.option_type,
			s.underlying_symbol_id, s.underlying_symbol, s.lot_size,
			s.tick_size, s.upper_circuit, s.lower_circuit, s.freeze_qty,
			s.description, s.ltp, s.open, s.high, s.low, s.close,
			s.vol, s.oi, s.bid, s.ask, s.bid_qty, s.ask_qty,
			s.cautionary_message_info, s.last_updated
		FROM watchlist_items wi
		JOIN stocks s ON s.id = wi.stock_id
		WHERE wi.watchlist_id = $1
		ORDER BY wi.added_at DESC
	`
	rows, err := r.db.Query(query, watchlistID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var items []models.WatchlistItem
	for rows.Next() {
		var item models.WatchlistItem
		var stock models.Stock
		err := rows.Scan(
			&item.ID, &item.WatchlistID, &item.StockID, &item.AddedAt,
			&stock.ID, &stock.ExchangeInstrumentID, &stock.Segment, &stock.InstrumentType,
			&stock.Symbol, &stock.DisplayName, &stock.CompanyName, &stock.ISIN, &stock.Series,
			&stock.Exchange, &stock.ContractExpiration, &stock.Strike, &stock.OptionType,
			&stock.UnderlyingSymbolID, &stock.UnderlyingSymbol, &stock.LotSize,
			&stock.TickSize, &stock.UpperCircuit, &stock.LowerCircuit, &stock.FreezeQty,
			&stock.Description, &stock.LTP, &stock.Open, &stock.High, &stock.Low, &stock.Close,
			&stock.Vol, &stock.OI, &stock.Bid, &stock.Ask, &stock.BidQty, &stock.AskQty,
			&stock.CautionaryMessageInfo, &stock.LastUpdated,
		)
		if err != nil {
			return nil, err
		}
		item.Stock = &stock
		items = append(items, item)
	}
	return items, nil
}

func (r *Repository) AddStock(watchlistID, stockID int) error {
	query := `
		INSERT INTO watchlist_items (watchlist_id, stock_id, added_at)
		VALUES ($1, $2, NOW())
	`
	_, err := r.db.Exec(query, watchlistID, stockID)
	return err
}

func (r *Repository) RemoveStock(watchlistID, stockID int) error {
	_, err := r.db.Exec(
		`DELETE FROM watchlist_items WHERE watchlist_id = $1 AND stock_id = $2`,
		watchlistID, stockID,
	)
	return err
}

func (r *Repository) StockExists(watchlistID, stockID int) (bool, error) {
	var count int
	err := r.db.QueryRow(
		`SELECT COUNT(*) FROM watchlist_items WHERE watchlist_id = $1 AND stock_id = $2`,
		watchlistID, stockID,
	).Scan(&count)
	return count > 0, err
}