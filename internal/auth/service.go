package auth

import (
	"errors"
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
	return &Service{repo: repo, jwtSecret: jwtSecret}
}

func (s *Service) Register(req *models.RegisterRequest) (*models.AuthResponse, error) {

	// 1. check if user already exists
	_, _, err := s.repo.GetUserByEmail(req.Email)
	if err == nil {
		return nil, errors.New("email already exists")
	}

	if err.Error() != "user not found" {
		return nil, err
	}

	// 2. hash password
	hash, err := bcrypt.GenerateFromPassword([]byte(req.Password), bcrypt.DefaultCost)
	if err != nil {
		return nil, err
	}

	// 3. create user
	user := &models.User{
		Name:  req.Name,
		Email: req.Email,
	}

	err = s.repo.CreateUser(user, string(hash))
	if err != nil {
		return nil, err
	}

	// 4. generate token
	token, err := s.generateToken(user.ID)
	if err != nil {
		return nil, err
	}

	return &models.AuthResponse{
		Token: token,
		User:  *user,
	}, nil
}

func (s *Service) Login(req *models.LoginRequest) (*models.AuthResponse, error) {
	user, passwordHash, err := s.repo.GetUserByEmail(req.Email)
	if err != nil {
		return nil, errors.New("invalid email or password")
	}

	// Password verify karo
	if err := bcrypt.CompareHashAndPassword([]byte(passwordHash), []byte(req.Password)); err != nil {
		return nil, errors.New("invalid email or password")
	}

	token, err := s.generateToken(user.ID)
	if err != nil {
		return nil, err
	}

	return &models.AuthResponse{Token: token, User: *user}, nil
}

func (s *Service) generateToken(userID int) (string, error) {
	claims := jwt.MapClaims{
		"user_id": userID,
		"exp":     time.Now().Add(24 * time.Hour).Unix(),
	}
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	return token.SignedString([]byte(s.jwtSecret))
}
