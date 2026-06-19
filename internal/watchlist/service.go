package watchlist

import (
	"errors"
	"strconv"
	"strings"
	"watchlist-backend/pkg/models"
)

type Service struct {
	repo *Repository
}

func NewService(repo *Repository) *Service {
	return &Service{repo: repo}
}

func (s *Service) CreateWatchlist(userID int, name string) (*models.Watchlist, error) {
	exists, err := s.repo.ExistsByName(userID, strings.TrimSpace(name))
	if err != nil {
		return nil, err
	}
	if exists {
		return nil, errors.New("watchlist with this name already exists")
	}
	w := &models.Watchlist{
		UserID: userID,
		Name:   name,
	}
	if err := s.repo.Create(w); err != nil {
		return nil, err
	}
	return w, nil
}

func (s *Service) GetWatchlists(userID int) ([]models.Watchlist, error) {
	return s.repo.GetAllByUserID(userID)
}

func (s *Service) DeleteWatchlist(userID, watchlistID int) error {
	w, err := s.repo.GetByID(watchlistID)
	if err != nil {
		return err
	}
	if w.UserID != userID {
		return errors.New("unauthorized")
	}
	return s.repo.Delete(watchlistID)
}

func (s *Service) GetStocks(userID, watchlistID int) ([]models.WatchlistItem, error) {
	w, err := s.repo.GetByID(watchlistID)
	if err != nil {
		return nil, err
	}
	if w.UserID != userID {
		return nil, errors.New("unauthorized")
	}
	return s.repo.GetStocks(watchlistID)
}

func (s *Service) AddStock(userID, watchlistID, stockID int) error {
	w, err := s.repo.GetByID(watchlistID)
	if err != nil {
		return err
	}
	if w.UserID != userID {
		return errors.New("unauthorized")
	}
	exists, err := s.repo.StockExists(watchlistID, stockID)
	if err != nil {
		return err
	}
	if exists {
		return errors.New("stock already in watchlist")
	}
	return s.repo.AddStock(watchlistID, stockID)
}

func (s *Service) RemoveStock(userID, watchlistID, stockID int) error {
	w, err := s.repo.GetByID(watchlistID)
	if err != nil {
		return err
	}
	if w.UserID != userID {
		return errors.New("unauthorized")
	}
	return s.repo.RemoveStock(watchlistID, stockID)
}

// String to Int helper
func ParseInt(s string) (int, error) {
	val, err := strconv.Atoi(s)
	if err != nil {
		return 0, errors.New("invalid id format")
	}
	return val, nil
}
