// Package auth maneja usuarios, contrasenas y tokens JWT (HS256).
package auth

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"ecommerce/internal/apperr"
)

const (
	RoleAdmin    = "admin"
	RoleCustomer = "customer"
)

// jwtHeader es fijo: solo se aceptan tokens HS256 (evita ataques de alg).
var jwtHeader = base64.RawURLEncoding.EncodeToString([]byte(`{"alg":"HS256","typ":"JWT"}`))

type claims struct {
	Sub   string `json:"sub"`
	Email string `json:"email"`
	Role  string `json:"role"`
	Iat   int64  `json:"iat"`
	Exp   int64  `json:"exp"`
}

type TokenManager struct {
	secret []byte
	ttl    time.Duration
	now    func() time.Time
}

func NewTokenManager(secret string, ttl time.Duration) *TokenManager {
	return &TokenManager{secret: []byte(secret), ttl: ttl, now: time.Now}
}

func (m *TokenManager) sign(data string) string {
	mac := hmac.New(sha256.New, m.secret)
	mac.Write([]byte(data))
	return base64.RawURLEncoding.EncodeToString(mac.Sum(nil))
}

// Issue genera un token firmado para el usuario.
func (m *TokenManager) Issue(u User) (string, time.Time, error) {
	now := m.now()
	exp := now.Add(m.ttl)
	payload, err := json.Marshal(claims{Sub: u.ID, Email: u.Email, Role: u.Role, Iat: now.Unix(), Exp: exp.Unix()})
	if err != nil {
		return "", time.Time{}, err
	}
	data := jwtHeader + "." + base64.RawURLEncoding.EncodeToString(payload)
	return data + "." + m.sign(data), exp, nil
}

// Parse valida firma y expiracion y devuelve la identidad.
func (m *TokenManager) Parse(token string) (Principal, error) {
	invalid := fmt.Errorf("%w: token invalido", apperr.ErrUnauthorized)
	parts := strings.Split(token, ".")
	if len(parts) != 3 || parts[0] != jwtHeader {
		return Principal{}, invalid
	}
	expected := m.sign(parts[0] + "." + parts[1])
	if !hmac.Equal([]byte(expected), []byte(parts[2])) {
		return Principal{}, invalid
	}
	raw, err := base64.RawURLEncoding.DecodeString(parts[1])
	if err != nil {
		return Principal{}, invalid
	}
	var c claims
	if err := json.Unmarshal(raw, &c); err != nil || c.Sub == "" {
		return Principal{}, invalid
	}
	if m.now().Unix() >= c.Exp {
		return Principal{}, fmt.Errorf("%w: token expirado", apperr.ErrUnauthorized)
	}
	return Principal{UserID: c.Sub, Email: c.Email, Role: c.Role}, nil
}
