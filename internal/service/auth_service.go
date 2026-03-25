package service

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/crm-system-new/crm-identity/internal/domain"
	"github.com/crm-system-new/crm-shared/pkg/audit"
	"github.com/crm-system-new/crm-shared/pkg/auth"
	"github.com/crm-system-new/crm-shared/pkg/outbox"
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
	repo         domain.UserRepository
	pool         *pgxpool.Pool
	outboxStore  outbox.Store
	auditLogger  *audit.Logger
	jwtManager   *auth.JWTManager
}

func NewAuthService(repo domain.UserRepository, pool *pgxpool.Pool, outboxStore outbox.Store, auditLogger *audit.Logger, jwtManager *auth.JWTManager) *AuthService {
	return &AuthService{
		repo:        repo,
		pool:        pool,
		outboxStore: outboxStore,
		auditLogger: auditLogger,
		jwtManager:  jwtManager,
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

	// Start transaction for atomic save + outbox + audit
	tx, err := s.pool.Begin(ctx)
	if err != nil {
		return nil, fmt.Errorf("begin tx: %w", err)
	}
	defer tx.Rollback(ctx)

	if err := s.repo.SaveInTx(ctx, tx, user); err != nil {
		return nil, fmt.Errorf("save user: %w", err)
	}

	// Convert domain events to outbox entries
	events := user.PullEvents()
	entries, err := outbox.FromDomainEvents(events, "crm.")
	if err != nil {
		return nil, fmt.Errorf("convert events to outbox: %w", err)
	}

	if err := s.outboxStore.InsertInTx(ctx, tx, entries); err != nil {
		return nil, fmt.Errorf("insert outbox entries: %w", err)
	}

	// Audit log
	changes, _ := json.Marshal(map[string]string{
		"email":      req.Email,
		"first_name": req.FirstName,
		"last_name":  req.LastName,
	})
	if err := s.auditLogger.LogInTx(ctx, tx, audit.LogEntry{
		Action:     "create",
		EntityType: "user",
		EntityID:   user.ID,
		UserID:     user.ID,
		Changes:    changes,
	}); err != nil {
		return nil, fmt.Errorf("audit log: %w", err)
	}

	if err := tx.Commit(ctx); err != nil {
		return nil, fmt.Errorf("commit tx: %w", err)
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

	// Update last login + publish auth event via outbox in a transaction
	tx, err := s.pool.Begin(ctx)
	if err != nil {
		return nil, fmt.Errorf("begin tx: %w", err)
	}
	defer tx.Rollback(ctx)

	if err := s.repo.UpdateInTx(ctx, tx, user); err != nil {
		return nil, fmt.Errorf("update user: %w", err)
	}

	events := user.PullEvents()
	entries, err := outbox.FromDomainEvents(events, "crm.")
	if err != nil {
		return nil, fmt.Errorf("convert events to outbox: %w", err)
	}

	if err := s.outboxStore.InsertInTx(ctx, tx, entries); err != nil {
		return nil, fmt.Errorf("insert outbox entries: %w", err)
	}

	if err := tx.Commit(ctx); err != nil {
		return nil, fmt.Errorf("commit tx: %w", err)
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
