package auth

import (
	"context"
	"errors"
	"fmt"
	"net/mail"
	"strings"
	"time"

	"ecommerce/internal/apperr"
	"ecommerce/internal/platform/database"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"golang.org/x/crypto/bcrypt"
)

type User struct {
	ID        string    `json:"id"`
	Email     string    `json:"email"`
	Name      string    `json:"name"`
	Role      string    `json:"role"`
	Active    bool      `json:"active"`
	CreatedAt time.Time `json:"created_at"`
}

// Principal es la identidad autenticada de una peticion.
type Principal struct {
	UserID string
	Email  string
	Role   string
}

func (p Principal) IsAdmin() bool { return p.Role == RoleAdmin }

type ctxKey struct{}

func WithPrincipal(ctx context.Context, p Principal) context.Context {
	return context.WithValue(ctx, ctxKey{}, p)
}

func FromContext(ctx context.Context) (Principal, bool) {
	p, ok := ctx.Value(ctxKey{}).(Principal)
	return p, ok
}

type RegisterInput struct {
	Email    string `json:"email"`
	Name     string `json:"name"`
	Password string `json:"password"`
}

type LoginResult struct {
	Token     string    `json:"token"`
	ExpiresAt time.Time `json:"expires_at"`
	User      User      `json:"user"`
}

// dummyHash se compara cuando el email no existe, para que login tarde igual
// y no revele que cuentas existen.
var dummyHash, _ = bcrypt.GenerateFromPassword([]byte("dummy-password-for-timing"), bcrypt.DefaultCost)

type Service struct {
	pool   *pgxpool.Pool
	tokens *TokenManager
}

func NewService(pool *pgxpool.Pool, tokens *TokenManager) *Service {
	return &Service{pool: pool, tokens: tokens}
}

func validatePassword(p string) error {
	if len(p) < 8 {
		return apperr.Validation("la contrasena debe tener al menos 8 caracteres")
	}
	if len(p) > 72 {
		return apperr.Validation("la contrasena no puede superar 72 caracteres")
	}
	return nil
}

func normalizeEmail(e string) (string, error) {
	e = strings.ToLower(strings.TrimSpace(e))
	addr, err := mail.ParseAddress(e)
	if err != nil || addr.Address != e {
		return "", apperr.Validation("email invalido")
	}
	return e, nil
}

const userColumns = `id::text, email, name, role, active, created_at`

func scanUser(row pgx.Row) (User, error) {
	var u User
	err := row.Scan(&u.ID, &u.Email, &u.Name, &u.Role, &u.Active, &u.CreatedAt)
	return u, database.MapError(err)
}

// Register crea una cuenta de cliente.
func (s *Service) Register(ctx context.Context, in RegisterInput) (User, error) {
	email, err := normalizeEmail(in.Email)
	if err != nil {
		return User{}, err
	}
	name := strings.TrimSpace(in.Name)
	if name == "" {
		return User{}, apperr.Validation("name es obligatorio")
	}
	if err := validatePassword(in.Password); err != nil {
		return User{}, err
	}
	return s.insert(ctx, email, name, in.Password, RoleCustomer)
}

func (s *Service) insert(ctx context.Context, email, name, password, role string) (User, error) {
	hash, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		return User{}, err
	}
	u, err := scanUser(s.pool.QueryRow(ctx,
		`INSERT INTO users (email, password_hash, name, role) VALUES ($1, $2, $3, $4) RETURNING `+userColumns,
		email, string(hash), name, role))
	if errors.Is(err, apperr.ErrConflict) {
		return User{}, fmt.Errorf("%w: el email ya esta registrado", apperr.ErrConflict)
	}
	return u, err
}

// Login valida credenciales y emite un token.
func (s *Service) Login(ctx context.Context, email, password string) (LoginResult, error) {
	fail := fmt.Errorf("%w: credenciales invalidas", apperr.ErrUnauthorized)
	email = strings.ToLower(strings.TrimSpace(email))

	var u User
	var hash string
	err := s.pool.QueryRow(ctx,
		`SELECT `+userColumns+`, password_hash FROM users WHERE email = $1`, email).
		Scan(&u.ID, &u.Email, &u.Name, &u.Role, &u.Active, &u.CreatedAt, &hash)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			_ = bcrypt.CompareHashAndPassword(dummyHash, []byte(password))
			return LoginResult{}, fail
		}
		return LoginResult{}, err
	}
	if bcrypt.CompareHashAndPassword([]byte(hash), []byte(password)) != nil || !u.Active {
		return LoginResult{}, fail
	}
	token, exp, err := s.tokens.Issue(u)
	if err != nil {
		return LoginResult{}, err
	}
	return LoginResult{Token: token, ExpiresAt: exp, User: u}, nil
}

// Authenticate valida el token y confirma que el usuario sigue activo.
func (s *Service) Authenticate(ctx context.Context, token string) (Principal, error) {
	p, err := s.tokens.Parse(token)
	if err != nil {
		return Principal{}, err
	}
	var active bool
	var role string
	err = s.pool.QueryRow(ctx, `SELECT active, role FROM users WHERE id = $1::uuid`, p.UserID).Scan(&active, &role)
	if err != nil || !active {
		return Principal{}, fmt.Errorf("%w: usuario no disponible", apperr.ErrUnauthorized)
	}
	p.Role = role // el rol vigente en BD manda sobre el del token
	return p, nil
}

func (s *Service) Get(ctx context.Context, id string) (User, error) {
	return scanUser(s.pool.QueryRow(ctx, `SELECT `+userColumns+` FROM users WHERE id = $1::uuid`, id))
}

// EnsureAdmin crea el administrador inicial si todavia no existe.
func (s *Service) EnsureAdmin(ctx context.Context, email, password string) (created bool, err error) {
	email, err = normalizeEmail(email)
	if err != nil {
		return false, err
	}
	var exists bool
	if err := s.pool.QueryRow(ctx, `SELECT EXISTS (SELECT 1 FROM users WHERE email = $1)`, email).Scan(&exists); err != nil {
		return false, err
	}
	if exists {
		return false, nil
	}
	_, err = s.insert(ctx, email, "Administrador", password, RoleAdmin)
	return err == nil, err
}
