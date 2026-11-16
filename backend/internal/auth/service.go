package auth

// Logica de negocio de autenticacion: registro, login y validacion de tokens.

import (
	"context"
	"errors"
	"fmt"
	"net/mail"
	"strings"

	"ecommerce/internal/apperr"

	"golang.org/x/crypto/bcrypt"
)

// dummyHash se compara cuando el email no existe, para que login tarde igual
// y no revele que cuentas existen.
var dummyHash, _ = bcrypt.GenerateFromPassword([]byte("dummy-password-for-timing"), bcrypt.DefaultCost)

type Service struct {
	users  *Repository
	tokens *TokenManager
}

func NewService(users *Repository, tokens *TokenManager) *Service {
	return &Service{users: users, tokens: tokens}
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
	return s.create(ctx, newUser{Email: email, Name: name, Role: RoleCustomer}, in.Password)
}

// create hashea la contrasena y guarda el usuario.
func (s *Service) create(ctx context.Context, n newUser, password string) (User, error) {
	hash, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		return User{}, err
	}
	n.PasswordHash = string(hash)
	u, err := s.users.Insert(ctx, n)
	if errors.Is(err, apperr.ErrConflict) {
		return User{}, fmt.Errorf("%w: el email ya esta registrado", apperr.ErrConflict)
	}
	return u, err
}

// Login valida credenciales y emite un token.
func (s *Service) Login(ctx context.Context, email, password string) (LoginResult, error) {
	fail := fmt.Errorf("%w: credenciales invalidas", apperr.ErrUnauthorized)
	email = strings.ToLower(strings.TrimSpace(email))

	u, hash, err := s.users.GetByEmail(ctx, email)
	if errors.Is(err, apperr.ErrNotFound) {
		_ = bcrypt.CompareHashAndPassword(dummyHash, []byte(password))
		return LoginResult{}, fail
	}
	if err != nil {
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
	u, err := s.users.Get(ctx, p.UserID)
	if err != nil || !u.Active {
		return Principal{}, fmt.Errorf("%w: usuario no disponible", apperr.ErrUnauthorized)
	}
	p.Role = u.Role // el rol vigente en BD manda sobre el del token
	return p, nil
}

func (s *Service) Get(ctx context.Context, id string) (User, error) {
	return s.users.Get(ctx, id)
}

// EnsureAdmin crea el administrador inicial si todavia no existe.
func (s *Service) EnsureAdmin(ctx context.Context, email, password string) (created bool, err error) {
	email, err = normalizeEmail(email)
	if err != nil {
		return false, err
	}
	exists, err := s.users.ExistsByEmail(ctx, email)
	if err != nil || exists {
		return false, err
	}
	_, err = s.create(ctx, newUser{Email: email, Name: "Administrador", Role: RoleAdmin}, password)
	return err == nil, err
}
