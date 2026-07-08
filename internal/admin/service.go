package admin

import (
	"errors"

	"watchlist-backend/internal/auth"
	"watchlist-backend/pkg/models"
)

type Service struct {
	authRepo *auth.Repository
}

func NewService(repo *auth.Repository) *Service {
	return &Service{
		authRepo: repo,
	}
}

// Revoke one device session
func (s *Service) RevokeUserSession(userID int, deviceType string) error {

	if deviceType != models.DeviceMobile &&
		deviceType != models.DeviceDesktop {
		return errors.New("invalid device type")
	}

	// Permanently block user
	if err := s.authRepo.BlockUser(userID); err != nil {
		return err
	}

	// Revoke current session
	return s.authRepo.RevokeAndDeleteSession(userID, deviceType)
}

// Permanently block user + revoke every session
func (s *Service) BlockUser(userID int) error {

	if err := s.authRepo.BlockUser(userID); err != nil {
		return err
	}

	return s.authRepo.RevokeAllSessions(userID)
}

// Revoke every session (without unblock)
func (s *Service) RevokeAllUserSessions(userID int) error {
	return s.authRepo.RevokeAllSessions(userID)
}

// Unblock user
func (s *Service) UnblockUser(userID int) error {
	return s.authRepo.UnblockUser(userID)
}