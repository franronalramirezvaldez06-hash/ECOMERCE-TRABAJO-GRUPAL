// Package config carga y valida la configuracion desde variables de entorno.
package config

import (
	"errors"
	"os"
	"strconv"
	"strings"
	"time"
)

const devJWTSecret = "dev-only-secret-change-me-dev-only-secret"

type Config struct {
	Env         string // development | production
	HTTPPort    string
	DatabaseURL string
	Currency    string

	ShippingCents         int64
	FreeShippingOverCents int64
	MaxPaymentCents       int64

	JWTSecret     string
	JWTTTL        time.Duration
	AdminEmail    string
	AdminPassword string

	AllowedOrigins []string
	TrustProxy     bool
	RateLimitRPS   float64
	RateLimitBurst int

	// StripeSecretKey activa la pasarela real de Stripe (reemplaza a la simulada).
	StripeSecretKey string
	// EnableSimulatedGateways registra pasarelas de prueba (stripe/paypal/mercadopago simuladas).
	EnableSimulatedGateways bool
}

func (c Config) IsProduction() bool { return c.Env == "production" }

func Load() (Config, error) {
	prod := strings.EqualFold(env("APP_ENV", "development"), "production")
	c := Config{
		Env:                     map[bool]string{true: "production", false: "development"}[prod],
		HTTPPort:                env("HTTP_PORT", "8080"),
		DatabaseURL:             env("DATABASE_URL", "postgres://ecommerce:ecommerce@localhost:5432/ecommerce?sslmode=disable"),
		Currency:                strings.ToUpper(env("CURRENCY", "PEN")),
		ShippingCents:           envInt("SHIPPING_CENTS", 500),
		FreeShippingOverCents:   envInt("FREE_SHIPPING_OVER_CENTS", 10000),
		MaxPaymentCents:         envInt("MAX_PAYMENT_CENTS", 500000),
		JWTSecret:               os.Getenv("JWT_SECRET"),
		JWTTTL:                  time.Duration(envInt("JWT_TTL_HOURS", 24)) * time.Hour,
		AdminEmail:              strings.ToLower(os.Getenv("ADMIN_EMAIL")),
		AdminPassword:           os.Getenv("ADMIN_PASSWORD"),
		TrustProxy:              envBool("TRUST_PROXY", false),
		RateLimitRPS:            envFloat("RATE_LIMIT_RPS", 20),
		RateLimitBurst:          int(envInt("RATE_LIMIT_BURST", 60)),
		StripeSecretKey:         os.Getenv("STRIPE_SECRET_KEY"),
		EnableSimulatedGateways: envBool("ENABLE_SIMULATED_GATEWAYS", !prod),
	}

	origins := os.Getenv("ALLOWED_ORIGINS")
	if origins == "" && !prod {
		origins = "*"
	}
	for _, o := range strings.Split(origins, ",") {
		if o = strings.TrimSpace(o); o != "" {
			c.AllowedOrigins = append(c.AllowedOrigins, o)
		}
	}

	if !prod {
		if c.JWTSecret == "" {
			c.JWTSecret = devJWTSecret
		}
		if c.AdminEmail == "" {
			c.AdminEmail, c.AdminPassword = "admin@ecommerce.local", "Admin12345!"
		}
	}
	return c, c.validate()
}

func (c Config) validate() error {
	var errs []error
	if len(c.JWTSecret) < 32 {
		errs = append(errs, errors.New("JWT_SECRET es obligatorio y debe tener al menos 32 caracteres"))
	}
	if c.IsProduction() {
		if c.JWTSecret == devJWTSecret {
			errs = append(errs, errors.New("JWT_SECRET no puede ser el valor de desarrollo"))
		}
		if c.StripeSecretKey == "" && !c.EnableSimulatedGateways {
			errs = append(errs, errors.New("en produccion configura STRIPE_SECRET_KEY (o ENABLE_SIMULATED_GATEWAYS=true)"))
		}
	}
	if c.AdminEmail != "" && len(c.AdminPassword) < 10 {
		errs = append(errs, errors.New("ADMIN_PASSWORD debe tener al menos 10 caracteres"))
	}
	return errors.Join(errs...)
}

func env(key, def string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return def
}

func envInt(key string, def int64) int64 {
	if v, err := strconv.ParseInt(os.Getenv(key), 10, 64); err == nil {
		return v
	}
	return def
}

func envFloat(key string, def float64) float64 {
	if v, err := strconv.ParseFloat(os.Getenv(key), 64); err == nil {
		return v
	}
	return def
}

func envBool(key string, def bool) bool {
	if v, err := strconv.ParseBool(os.Getenv(key)); err == nil {
		return v
	}
	return def
}
