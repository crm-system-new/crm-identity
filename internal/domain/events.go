package domain

import "github.com/crm-system-new/crm-shared/pkg/ddd"

// UserRegistered is raised when a new user is created.
type UserRegistered struct {
	ddd.BaseEvent
	Email     string `json:"email"`
	FirstName string `json:"first_name"`
	LastName  string `json:"last_name"`
}

// UserAuthenticated is raised on successful login.
type UserAuthenticated struct {
	ddd.BaseEvent
}
