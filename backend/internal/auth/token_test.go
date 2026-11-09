package auth

import (
	"strings"
	"testing"
	"time"
)

func TestTokenRoundTrip(t *testing.T) {
	m := NewTokenManager("secret-secret-secret-secret-secret", time.Hour)
	tok, _, err := m.Issue(User{ID: "u1", Email: "a@b.com", Role: RoleAdmin})
	if err != nil {
		t.Fatal(err)
	}
	p, err := m.Parse(tok)
	if err != nil || p.UserID != "u1" || !p.IsAdmin() || p.Email != "a@b.com" {
		t.Fatalf("principal inesperado: %+v err=%v", p, err)
	}
}

func TestTokenRejectsTampering(t *testing.T) {
	m := NewTokenManager("secret-secret-secret-secret-secret", time.Hour)
	tok, _, _ := m.Issue(User{ID: "u1", Role: RoleCustomer})
	parts := strings.Split(tok, ".")

	forged := parts[0] + "." + parts[1] + "x." + parts[2]
	if _, err := m.Parse(forged); err == nil {
		t.Fatal("debio rechazar payload alterado")
	}
	other := NewTokenManager("otro-secreto-otro-secreto-otro-secreto", time.Hour)
	if _, err := other.Parse(tok); err == nil {
		t.Fatal("debio rechazar firma de otro secreto")
	}
	if _, err := m.Parse("a.b"); err == nil {
		t.Fatal("debio rechazar formato invalido")
	}
}

func TestTokenExpires(t *testing.T) {
	m := NewTokenManager("secret-secret-secret-secret-secret", time.Minute)
	now := time.Now()
	m.now = func() time.Time { return now }
	tok, _, _ := m.Issue(User{ID: "u1", Role: RoleCustomer})
	m.now = func() time.Time { return now.Add(2 * time.Minute) }
	if _, err := m.Parse(tok); err == nil {
		t.Fatal("debio rechazar token expirado")
	}
}

func TestPasswordPolicy(t *testing.T) {
	if err := validatePassword("corta"); err == nil {
		t.Fatal("debio rechazar contrasena corta")
	}
	if err := validatePassword(strings.Repeat("a", 80)); err == nil {
		t.Fatal("debio rechazar contrasena de mas de 72 bytes")
	}
	if err := validatePassword("una-clave-razonable"); err != nil {
		t.Fatal(err)
	}
}
