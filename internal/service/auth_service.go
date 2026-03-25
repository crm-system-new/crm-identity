package service

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/crm-system-new/crm-identity/internal/domain"
	"github.com/crm-system-new/crm-shared/pkg/auth"
	"github.com/crm-system-new/crm-shared/pkg/messaging"
)

type RegisterRequest struct {
	Email     string `json:"email"`
	Password  string `json:"password"`
	FirstName string `json:"first_name"`
	LastName  string `json:"last_name"`
}

type RegisterResponse struct {
	ID    string `json:"id"`
	Email string `json:"email"`
}

type LoginRequest struct {
	Email    string `json:"email"`
	Password string `json:"password"`
}

type AuthService struct {
	repo       domain.UserRepository
	publisher  messaging.EventPublisher
	jwtManager *auth.JWTManager
}

func NewAuthService(repo domain.UserRepository, publisher messaging.EventPublisher, jwtManager *auth.JWTManager) *AuthService {
	return &AuthService{
		repo:       repo,
		publisher:  publisher,
		jwtManager: jwtManager,
	}
}

func (s *AuthService) Register(ctx context.Context, req RegisterRequest) (*RegisterResponse, error) {
	existing, _ := s.repo.GetByEmail(ctx, req.Email)
	if existing != nil {
		return nil, domain.ErrEmailAlreadyExists
	}

	hash, err := domain.HashPassword(req.Password)
	if err != nil {
		return nil, fmt.Errorf("hash password: %w", err)
	}

	user, err := domain.NewUser(req.Email, hash, req.FirstName, req.LastName)
	if err != nil {
		return nil, err
	}

	if err := s.repo.Save(ctx, user); err != nil {
		return nil, fmt.Errorf("save user: %w", err)
	}

	// Publish domain events
	for _, event := range user.PullEvents() {
		data, _ := json.Marshal(event)
		s.publisher.Publish(ctx, "crm."+event.EventType(), data)
	}

	return &RegisterResponse{
		ID:    user.ID,
		Email: user.Email,
	}, nil
}

func (s *AuthService) Login(ctx context.Context, req LoginRequest) (*auth.TokenPair, error) {
	user, err := s.repo.GetByEmail(ctx, req.Email)
	if err != nil {
		return nil, domain.ErrInvalidCredentials
	}

	if !domain.CheckPassword(user.PasswordHash, req.Password) {
		return nil, domain.ErrInvalidCredentials
	}

	if err := user.Authenticate(); err != nil {
		return nil, err
	}

	// Collect role names
	roleNames := make([]string, len(user.Roles))
	for i, r := range user.Roles {
		roleNames[i] = r.Name
	}

	tokenPair, err := s.jwtManager.GenerateTokenPair(user.ID, user.Email, roleNames)
	if err != nil {
		return nil, fmt.Errorf("generate tokens: %w", err)
	}

	// Update last login
	_ = s.repo.Update(ctx, user)

	// Publish events
	for _, event := range user.PullEvents() {
		data, _ := json.Marshal(event)
		s.publisher.Publish(ctx, "crm."+event.EventType(), data)
	}

	return tokenPair, nil
}

func (s *AuthService) RefreshToken(ctx context.Context, refreshToken string) (*auth.TokenPair, error) {
	claims, err := s.jwtManager.ValidateToken(refreshToken)
	if err != nil {
		return nil, fmt.Errorf("invalid refresh token: %w", err)
	}

	user, err := s.repo.GetByID(ctx, claims.UserID)
	if err != nil {
		return nil, fmt.Errorf("user not found: %w", err)
	}

	if user.Status != domain.UserStatusActive {
		return nil, domain.ErrInvalidCredentials
	}

	roleNames := make([]string, len(user.Roles))
	for i, r := range user.Roles {
		roleNames[i] = r.Name
	}

	_ = time.Now() // acknowledge time import
	return s.jwtManager.GenerateTokenPair(user.ID, user.Email, roleNames)
}
