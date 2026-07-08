package auth

import (
	"crypto/rand"
	"encoding/hex"
	"errors"
	"strings"
	"time"

	"watchlist-backend/pkg/models"

	"github.com/golang-jwt/jwt/v5"
	"golang.org/x/crypto/bcrypt"
)

type Service struct {
	repo      *Repository
	jwtSecret string
}

func NewService(repo *Repository, jwtSecret string) *Service {
	return &Service{
		repo:      repo,
		jwtSecret: jwtSecret,
	}
}

func (s *Service) Register(req *models.RegisterRequest, userAgent string) (*models.AuthResponse, error) {

	req.Email = strings.ToLower(strings.TrimSpace(req.Email))
	req.Name = strings.TrimSpace(req.Name)

	_, _, err := s.repo.GetUserByEmail(req.Email)

	if err == nil {
		return nil, errors.New("email already exists")
	}

	hash, err := bcrypt.GenerateFromPassword(
		[]byte(req.Password),
		bcrypt.DefaultCost,
	)

	if err != nil {
		return nil, err
	}

	user := &models.User{
		Name:  req.Name,
		Email: req.Email,
	}

	err = s.repo.CreateUser(
		user,
		string(hash),
	)

	if err != nil {
		return nil, err
	}

	return s.issueSessionAndToken(
		user,
		req.DeviceType,
		userAgent,
	)
}

func (s *Service) Login(req *models.LoginRequest, userAgent string) (*models.AuthResponse, error) {

	req.Email = strings.ToLower(strings.TrimSpace(req.Email))

	user, passwordHash, err := s.repo.GetUserByEmail(req.Email)

	if err != nil {
		return nil, errors.New("invalid email or password")
	}

	// permanent blacklist check
	if user.IsBlocked {
		return nil, errors.New("your account has been blocked by admin")
	}

	err = bcrypt.CompareHashAndPassword(
		[]byte(passwordHash),
		[]byte(req.Password),
	)

	if err != nil {
		return nil, errors.New("invalid email or password")
	}

	return s.issueSessionAndToken(
		user,
		req.DeviceType,
		userAgent,
	)
}

func (s *Service) Logout(userID int, deviceType string) error {

	return s.repo.RevokeAndDeleteSession(
		userID,
		deviceType,
	)
}

func (s *Service) ListSessions(userID int) ([]models.SessionInfo, error) {

	sessions, err := s.repo.GetUserSessions(userID)

	if err != nil {
		return nil, err
	}

	result := make([]models.SessionInfo, 0)

	for _, session := range sessions {

		result = append(result, models.SessionInfo{
			DeviceType: session.DeviceType,
			DeviceInfo: session.DeviceInfo,
			CreatedAt:  session.CreatedAt,
			LastSeenAt: session.LastSeenAt,
		})
	}

	return result, nil
}

func (s *Service) RevokeSession(userID int, deviceType string) error {

	if deviceType != models.DeviceMobile &&
		deviceType != models.DeviceDesktop {

		return errors.New("invalid device type")
	}

	return s.repo.RevokeAndDeleteSession(
		userID,
		deviceType,
	)
}

func (s *Service) issueSessionAndToken(
	user *models.User,
	deviceType string,
	userAgent string,
) (*models.AuthResponse, error) {

	if deviceType == "" {
		deviceType = detectDeviceType(userAgent)
	}

	sessionID, err := generateSessionID()

	if err != nil {
		return nil, err
	}

	err = s.repo.UpsertSession(
		user.ID,
		deviceType,
		sessionID,
		userAgent,
	)

	if err != nil {
		return nil, err
	}

	token, err := s.generateToken(
		user.ID,
		sessionID,
		deviceType,
	)

	if err != nil {
		return nil, err
	}

	return &models.AuthResponse{
		Token: token,
		User:  *user,
	}, nil
}

func detectDeviceType(userAgent string) string {

	ua := strings.ToLower(userAgent)

	keywords := []string{
		"mobile",
		"android",
		"iphone",
		"ipad",
	}

	for _, k := range keywords {

		if strings.Contains(ua, k) {
			return models.DeviceMobile
		}
	}

	return models.DeviceDesktop
}

func generateSessionID() (string, error) {

	b := make([]byte, 16)

	_, err := rand.Read(b)

	if err != nil {
		return "", err
	}

	return hex.EncodeToString(b), nil
}

func (s *Service) generateToken(
	userID int,
	sessionID string,
	deviceType string,
) (string, error) {

	claims := jwt.MapClaims{

		"user_id": userID,

		"sid": sessionID,

		"jti": sessionID,

		"device_type": deviceType,

		"exp": time.Now().
			Add(24 * time.Hour).
			Unix(),
	}

	token := jwt.NewWithClaims(
		jwt.SigningMethodHS256,
		claims,
	)

	return token.SignedString(
		[]byte(s.jwtSecret),
	)
}

// Admin block
func (s *Service) BlockUser(userID int) error {

	err := s.repo.BlockUser(userID)

	if err != nil {
		return err
	}

	return s.repo.RevokeAllSessions(userID)
}

func (s *Service) UnblockUser(userID int) error {

	return s.repo.UnblockUser(userID)
}