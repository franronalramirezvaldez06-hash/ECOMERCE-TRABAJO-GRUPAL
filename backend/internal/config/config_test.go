package config

import "testing"

func TestDefaultCurrencyIsSoles(t *testing.T) {
	t.Setenv("APP_ENV", "development")
	t.Setenv("CURRENCY", "")
	t.Setenv("JWT_SECRET", "")
	t.Setenv("ADMIN_EMAIL", "")
	t.Setenv("ADMIN_PASSWORD", "")
	cfg, err := Load()
	if err != nil {
		t.Fatal(err)
	}
	if cfg.Currency != "PEN" {
		t.Fatalf("moneda predeterminada = %q; se esperaba PEN", cfg.Currency)
	}
}
