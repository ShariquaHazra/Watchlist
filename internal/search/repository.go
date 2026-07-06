package search

import (
	"database/sql"
	"watchlist-backend/pkg/models"
)

type Repository struct {
	db *sql.DB
}

func NewRepository(db *sql.DB) *Repository {
	return &Repository{db: db}
}

func (r *Repository) SearchStocks(query string) ([]models.Stock, error) {

    search := query
    likeQuery := "%" + query + "%"

    rows, err := r.db.Query(`
        SELECT id, symbol, company_name, exchange, ltp, last_updated
        FROM stocks
        WHERE
            symbol ILIKE $1
            OR company_name ILIKE $1
        ORDER BY
            CASE
                WHEN LOWER(symbol) = LOWER($2) THEN 1
                WHEN LOWER(symbol) LIKE LOWER($2 || '%') THEN 2
                WHEN LOWER(company_name) LIKE LOWER($2 || '%') THEN 3
                ELSE 4
            END,
            symbol
        LIMIT 20
    `, likeQuery, search)//values passed
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var result []models.Stock

	for rows.Next() {
		var s models.Stock
		err := rows.Scan(
			&s.ID,
			&s.Symbol,
			&s.CompanyName,
			&s.Exchange,
			&s.LTP,
			&s.LastUpdated,
		)

		if err != nil {
			return nil, err
		}

		result = append(result, s)
	}

	return result, nil
}