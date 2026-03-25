CREATE TABLE IF NOT EXISTS roles (
    id          VARCHAR(36) PRIMARY KEY,
    name        VARCHAR(50) NOT NULL UNIQUE,
    description TEXT,
    created_at  TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE TABLE IF NOT EXISTS user_roles (
    user_id VARCHAR(36) NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    role_id VARCHAR(36) NOT NULL REFERENCES roles(id) ON DELETE CASCADE,
    PRIMARY KEY (user_id, role_id)
);

INSERT INTO roles (id, name, description) VALUES
    ('role-admin', 'admin', 'System administrator'),
    ('role-sales', 'sales_agent', 'Sales team member'),
    ('role-mktg', 'marketing_manager', 'Marketing team member'),
    ('role-support', 'support_agent', 'Support team member');
