package postgres

import (
	"context"
	"fmt"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/crm-system-new/crm-identity/internal/domain"
	"github.com/crm-system-new/crm-shared/pkg/ddd"
	sharedpg "github.com/crm-system-new/crm-shared/pkg/postgres"
)

type UserRepository struct {
	pool *pgxpool.Pool
}

func NewUserRepository(pool *pgxpool.Pool) *UserRepository {
	return &UserRepository{pool: pool}
}

func (r *UserRepository) GetByID(ctx context.Context, id string) (*domain.User, error) {
	query := `SELECT id, email, password_hash, first_name, last_name, status, last_login_at, version, created_at, updated_at
		FROM users WHERE id = $1`

	user := &domain.User{}
	err := r.pool.QueryRow(ctx, query, id).Scan(
		&user.ID, &user.Email, &user.PasswordHash,
		&user.FirstName, &user.LastName, &user.Status,
		&user.LastLoginAt, &user.Version, &user.CreatedAt, &user.UpdatedAt,
	)
	if err != nil {
		if err == pgx.ErrNoRows {
			return nil, ddd.ErrNotFound{Entity: "User", ID: id}
		}
		return nil, fmt.Errorf("query user by id: %w", err)
	}

	roles, err := r.getUserRoles(ctx, id)
	if err != nil {
		return nil, err
	}
	user.Roles = roles

	return user, nil
}

func (r *UserRepository) GetByEmail(ctx context.Context, email string) (*domain.User, error) {
	query := `SELECT id, email, password_hash, first_name, last_name, status, last_login_at, version, created_at, updated_at
		FROM users WHERE email = $1`

	user := &domain.User{}
	err := r.pool.QueryRow(ctx, query, email).Scan(
		&user.ID, &user.Email, &user.PasswordHash,
		&user.FirstName, &user.LastName, &user.Status,
		&user.LastLoginAt, &user.Version, &user.CreatedAt, &user.UpdatedAt,
	)
	if err != nil {
		if err == pgx.ErrNoRows {
			return nil, ddd.ErrNotFound{Entity: "User", ID: email}
		}
		return nil, fmt.Errorf("query user by email: %w", err)
	}

	roles, err := r.getUserRoles(ctx, user.ID)
	if err != nil {
		return nil, err
	}
	user.Roles = roles

	return user, nil
}

func (r *UserRepository) Save(ctx context.Context, user *domain.User) error {
	query := `INSERT INTO users (id, email, password_hash, first_name, last_name, status, last_login_at, version, created_at, updated_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10)`

	_, err := r.pool.Exec(ctx, query,
		user.ID, user.Email, user.PasswordHash,
		user.FirstName, user.LastName, user.Status,
		user.LastLoginAt, user.Version, user.CreatedAt, user.UpdatedAt,
	)
	if err != nil {
		return fmt.Errorf("insert user: %w", err)
	}
	return nil
}

func (r *UserRepository) SaveInTx(ctx context.Context, tx pgx.Tx, user *domain.User) error {
	query := `INSERT INTO users (id, email, password_hash, first_name, last_name, status, last_login_at, version, created_at, updated_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10)`

	_, err := tx.Exec(ctx, query,
		user.ID, user.Email, user.PasswordHash,
		user.FirstName, user.LastName, user.Status,
		user.LastLoginAt, user.Version, user.CreatedAt, user.UpdatedAt,
	)
	if err != nil {
		return fmt.Errorf("insert user in tx: %w", err)
	}
	return nil
}

func (r *UserRepository) Update(ctx context.Context, user *domain.User) error {
	oldVersion := user.Version
	user.IncrementVersion()

	query := `UPDATE users SET email=$1, password_hash=$2, first_name=$3, last_name=$4,
		status=$5, last_login_at=$6, updated_at=$7, version=$8
		WHERE id=$9 AND version=$10`

	return sharedpg.ExecWithOptimisticLockPool(ctx, r.pool, query,
		user.Email, user.PasswordHash, user.FirstName, user.LastName,
		user.Status, user.LastLoginAt, user.UpdatedAt, user.Version,
		user.ID, oldVersion,
	)
}

func (r *UserRepository) UpdateInTx(ctx context.Context, tx pgx.Tx, user *domain.User) error {
	oldVersion := user.Version
	user.IncrementVersion()

	query := `UPDATE users SET email=$1, password_hash=$2, first_name=$3, last_name=$4,
		status=$5, last_login_at=$6, updated_at=$7, version=$8
		WHERE id=$9 AND version=$10`

	return sharedpg.ExecWithOptimisticLock(ctx, tx, query,
		user.Email, user.PasswordHash, user.FirstName, user.LastName,
		user.Status, user.LastLoginAt, user.UpdatedAt, user.Version,
		user.ID, oldVersion,
	)
}

func (r *UserRepository) List(ctx context.Context, limit, offset int) ([]*domain.User, int, error) {
	var total int
	err := r.pool.QueryRow(ctx, `SELECT COUNT(*) FROM users`).Scan(&total)
	if err != nil {
		return nil, 0, fmt.Errorf("count users: %w", err)
	}

	query := `SELECT id, email, password_hash, first_name, last_name, status, last_login_at, version, created_at, updated_at
		FROM users ORDER BY created_at DESC LIMIT $1 OFFSET $2`

	rows, err := r.pool.Query(ctx, query, limit, offset)
	if err != nil {
		return nil, 0, fmt.Errorf("list users: %w", err)
	}
	defer rows.Close()

	var users []*domain.User
	for rows.Next() {
		user := &domain.User{}
		if err := rows.Scan(
			&user.ID, &user.Email, &user.PasswordHash,
			&user.FirstName, &user.LastName, &user.Status,
			&user.LastLoginAt, &user.Version, &user.CreatedAt, &user.UpdatedAt,
		); err != nil {
			return nil, 0, fmt.Errorf("scan user: %w", err)
		}
		users = append(users, user)
	}

	return users, total, nil
}

func (r *UserRepository) getUserRoles(ctx context.Context, userID string) ([]domain.Role, error) {
	query := `SELECT r.id, r.name, r.description FROM roles r
		JOIN user_roles ur ON r.id = ur.role_id
		WHERE ur.user_id = $1`

	rows, err := r.pool.Query(ctx, query, userID)
	if err != nil {
		return nil, fmt.Errorf("query user roles: %w", err)
	}
	defer rows.Close()

	var roles []domain.Role
	for rows.Next() {
		var role domain.Role
		if err := rows.Scan(&role.ID, &role.Name, &role.Description); err != nil {
			return nil, fmt.Errorf("scan role: %w", err)
		}
		roles = append(roles, role)
	}

	return roles, nil
}
