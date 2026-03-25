package domain

import "context"

// UserRepository defines persistence operations for the User aggregate.
type UserRepository interface {
	GetByID(ctx context.Context, id string) (*User, error)
	GetByEmail(ctx context.Context, email string) (*User, error)
	Save(ctx context.Context, user *User) error
	Update(ctx context.Context, user *User) error
	List(ctx context.Context, limit, offset int) ([]*User, int, error)
}
