package auth

// Capa de datos de usuarios: unico lugar de auth que habla con PostgreSQL.

import (
	"context"

	"ecommerce/internal/platform/database"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

// newUser son los datos para insertar un usuario (la contrasena ya viene hasheada).
type newUser struct {
	Email        string
	Name         string
	PasswordHash string
	Role         string
}

type Repository struct{ pool *pgxpool.Pool }

func NewRepository(pool *pgxpool.Pool) *Repository { return &Repository{pool: pool} }

const userColumns = `id::text, email, name, role, active, created_at`

func scanUser(row pgx.Row) (User, error) {
	var u User
	err := row.Scan(&u.ID, &u.Email, &u.Name, &u.Role, &u.Active, &u.CreatedAt)
	return u, database.MapError(err)
}

func (r *Repository) Insert(ctx context.Context, n newUser) (User, error) {
	return scanUser(r.pool.QueryRow(ctx,
		`INSERT INTO users (email, password_hash, name, role) VALUES ($1, $2, $3, $4) RETURNING `+userColumns,
		n.Email, n.PasswordHash, n.Name, n.Role))
}

func (r *Repository) Get(ctx context.Context, id string) (User, error) {
	return scanUser(r.pool.QueryRow(ctx, `SELECT `+userColumns+` FROM users WHERE id = $1::uuid`, id))
}

// GetByEmail devuelve el usuario y el hash de su contrasena.
func (r *Repository) GetByEmail(ctx context.Context, email string) (User, string, error) {
	var u User
	var hash string
	err := r.pool.QueryRow(ctx,
		`SELECT `+userColumns+`, password_hash FROM users WHERE email = $1`, email).
		Scan(&u.ID, &u.Email, &u.Name, &u.Role, &u.Active, &u.CreatedAt, &hash)
	return u, hash, database.MapError(err)
}

func (r *Repository) ExistsByEmail(ctx context.Context, email string) (bool, error) {
	var exists bool
	err := r.pool.QueryRow(ctx, `SELECT EXISTS (SELECT 1 FROM users WHERE email = $1)`, email).Scan(&exists)
	return exists, err
}
