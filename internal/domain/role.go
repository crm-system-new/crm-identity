package domain

// Role represents a user role in the system.
type Role struct {
	ID          string
	Name        string
	Description string
}

// Predefined role names.
const (
	RoleAdmin            = "admin"
	RoleSalesAgent       = "sales_agent"
	RoleMarketingManager = "marketing_manager"
	RoleSupportAgent     = "support_agent"
)
