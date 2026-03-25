package service

import (
	"context"
	"fmt"

	"github.com/crm-system-new/crm-identity/internal/domain"
)

type UpdateProfileRequest struct {
	FirstName string `json:"first_name"`
	LastName  string `json:"last_name"`
}

type ChangePasswordRequest struct {
	OldPassword string `json:"old_password"`
	NewPassword string `json:"new_password"`
}

type UserResponse struct {
	ID        string `json:"id"`
	Email     string `json:"email"`
	FirstName string `json:"first_name"`
	LastName  string `json:"last_name"`
	Status    string `json:"status"`
}

type UserService struct {
	repo domain.UserRepository
}

func NewUserService(repo domain.UserRepository) *UserService {
	return &UserService{repo: repo}
}

func (s *UserService) GetUser(ctx context.Context, id string) (*UserResponse, error) {
	user, err := s.repo.GetByID(ctx, id)
	if err != nil {
		return nil, err
	}
	return toUserResponse(user), nil
}

func (s *UserService) UpdateProfile(ctx context.Context, id string, req UpdateProfileRequest) (*UserResponse, error) {
	user, err := s.repo.GetByID(ctx, id)
	if err != nil {
		return nil, err
	}

	if err := user.UpdateProfile(req.FirstName, req.LastName); err != nil {
		return nil, err
	}

	if err := s.repo.Update(ctx, user); err != nil {
		return nil, fmt.Errorf("update user: %w", err)
	}

	return toUserResponse(user), nil
}

func (s *UserService) ChangePassword(ctx context.Context, id string, req ChangePasswordRequest) error {
	user, err := s.repo.GetByID(ctx, id)
	if err != nil {
		return err
	}

	if !domain.CheckPassword(user.PasswordHash, req.OldPassword) {
		return domain.ErrInvalidCredentials
	}

	hash, err := domain.HashPassword(req.NewPassword)
	if err != nil {
		return fmt.Errorf("hash password: %w", err)
	}

	user.ChangePassword(hash)
	return s.repo.Update(ctx, user)
}

func (s *UserService) ListUsers(ctx context.Context, limit, offset int) ([]*UserResponse, int, error) {
	users, total, err := s.repo.List(ctx, limit, offset)
	if err != nil {
		return nil, 0, err
	}

	responses := make([]*UserResponse, len(users))
	for i, u := range users {
		responses[i] = toUserResponse(u)
	}
	return responses, total, nil
}

func toUserResponse(u *domain.User) *UserResponse {
	return &UserResponse{
		ID:        u.ID,
		Email:     u.Email,
		FirstName: u.FirstName,
		LastName:  u.LastName,
		Status:    string(u.Status),
	}
}
