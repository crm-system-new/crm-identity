package domain

import (
	"time"

	"github.com/crm-system-new/crm-shared/pkg/ddd"
)

type UserStatus string

const (
	UserStatusActive   UserStatus = "active"
	UserStatusInactive UserStatus = "inactive"
	UserStatusLocked   UserStatus = "locked"
)

// User is the aggregate root for the Identity Bounded Context.
type User struct {
	ddd.AggregateRoot
	Email        string
	PasswordHash string
	FirstName    string
	LastName     string
	Status       UserStatus
	Roles        []Role
	LastLoginAt  *time.Time
}

// NewUser creates a new User aggregate and raises a UserRegistered event.
func NewUser(email, passwordHash, firstName, lastName string) (*User, error) {
	if email == "" {
		return nil, ddd.ErrValidation{Field: "email", Message: "email is required"}
	}
	if passwordHash == "" {
		return nil, ddd.ErrValidation{Field: "password", Message: "password is required"}
	}
	if firstName == "" {
		return nil, ddd.ErrValidation{Field: "first_name", Message: "first name is required"}
	}
	if lastName == "" {
		return nil, ddd.ErrValidation{Field: "last_name", Message: "last name is required"}
	}

	u := &User{
		AggregateRoot: ddd.NewAggregateRoot(),
		Email:         email,
		PasswordHash:  passwordHash,
		FirstName:     firstName,
		LastName:      lastName,
		Status:        UserStatusActive,
	}

	u.RaiseEvent(UserRegistered{
		BaseEvent: ddd.NewBaseEvent("identity.user.registered", u.ID),
		Email:     email,
		FirstName: firstName,
		LastName:  lastName,
	})

	return u, nil
}

// Authenticate marks a successful login and raises a UserAuthenticated event.
func (u *User) Authenticate() error {
	if u.Status != UserStatusActive {
		return ddd.ErrValidation{Field: "status", Message: "user account is not active"}
	}
	now := time.Now().UTC()
	u.LastLoginAt = &now
	u.RaiseEvent(UserAuthenticated{
		BaseEvent: ddd.NewBaseEvent("identity.user.authenticated", u.ID),
	})
	return nil
}

// Lock prevents the user from logging in.
func (u *User) Lock() error {
	if u.Status == UserStatusLocked {
		return ddd.ErrValidation{Field: "status", Message: "user is already locked"}
	}
	u.Status = UserStatusLocked
	u.IncrementVersion()
	return nil
}

// Activate reactivates a locked or inactive user.
func (u *User) Activate() error {
	if u.Status == UserStatusActive {
		return ddd.ErrValidation{Field: "status", Message: "user is already active"}
	}
	u.Status = UserStatusActive
	u.IncrementVersion()
	return nil
}

// UpdateProfile updates the user's name fields.
func (u *User) UpdateProfile(firstName, lastName string) error {
	if firstName == "" {
		return ddd.ErrValidation{Field: "first_name", Message: "first name is required"}
	}
	if lastName == "" {
		return ddd.ErrValidation{Field: "last_name", Message: "last name is required"}
	}
	u.FirstName = firstName
	u.LastName = lastName
	u.IncrementVersion()
	return nil
}

// ChangePassword updates the password hash.
func (u *User) ChangePassword(newPasswordHash string) {
	u.PasswordHash = newPasswordHash
	u.IncrementVersion()
}
