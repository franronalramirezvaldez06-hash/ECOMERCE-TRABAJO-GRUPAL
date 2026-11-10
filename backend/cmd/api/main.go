// Comando api: punto de entrada del monolito. Aqui se compone todo.
package main

import (
	"context"
	"errors"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"ecommerce/internal/auth"
	"ecommerce/internal/catalog"
	"ecommerce/internal/config"
	"ecommerce/internal/coupon"
	"ecommerce/internal/httpapi"
	"ecommerce/internal/inventory"
	"ecommerce/internal/payment"
	"ecommerce/internal/platform/database"
	"ecommerce/internal/pricing"
	"ecommerce/internal/sales"
)

func main() {
	log := slog.New(slog.NewJSONHandler(os.Stdout, nil))
	if err := run(log); err != nil {
		log.Error("fallo fatal", "error", err)
		os.Exit(1)
	}
}

func run(log *slog.Logger) error {
	cfg, err := config.Load()
	if err != nil {
		return err
	}
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	pool, err := database.Connect(ctx, cfg.DatabaseURL, log)
	if err != nil {
		return err
	}
	defer pool.Close()
	if err := database.Migrate(ctx, pool, log); err != nil {
		return err
	}

	authSvc := auth.NewService(pool, auth.NewTokenManager(cfg.JWTSecret, cfg.JWTTTL))
	if cfg.AdminEmail != "" {
		created, err := authSvc.EnsureAdmin(ctx, cfg.AdminEmail, cfg.AdminPassword)
		if err != nil {
			return err
		}
		if created {
			log.Info("administrador inicial creado", "email", cfg.AdminEmail)
		}
	}
	if !cfg.IsProduction() {
		log.Warn("modo desarrollo: no uses esta configuracion en produccion", "env", cfg.Env)
	}

	coupons := coupon.NewStore(pool)

	// Decorator (precios): subtotal -> cupon -> envio -> impuesto.
	calculator := checkoutCalculator(cfg, coupons)

	// Decorator (pasarelas): logging -> reintentos -> timeout -> limite de monto.
	decorate := func(g payment.Gateway) payment.Gateway {
		return payment.Decorate(g,
			payment.WithLogging(log),
			payment.WithRetry(3, 200*time.Millisecond),
			payment.WithTimeout(10*time.Second),
			payment.WithAmountLimit(cfg.MaxPaymentCents),
		)
	}
	var gws []payment.Gateway
	registered := map[string]bool{}
	if cfg.StripeSecretKey != "" {
		gws = append(gws, decorate(payment.NewStripe(cfg.StripeSecretKey)))
		registered["stripe"] = true
		log.Info("pasarela real habilitada", "gateway", "stripe")
	}
	if cfg.EnableSimulatedGateways {
		for _, name := range []string{"stripe", "paypal", "mercadopago"} {
			if !registered[name] {
				gws = append(gws, decorate(payment.NewSimulated(name)))
			}
		}
		log.Warn("pasarelas SIMULADAS habilitadas (no cobran dinero real)")
	}
	gateways := payment.NewRegistry(gws...)

	inv := inventory.NewStore(pool)
	cat := catalog.NewService(catalog.NewRepository(pool), inv)
	orders := sales.NewStore(pool)
	api := &httpapi.API{
		Auth:      authSvc,
		Catalog:   cat,
		Inventory: inv,
		Coupons:   coupons,
		Sales:     sales.NewService(pool, orders, cat, calculator, gateways, log),
		Orders:    orders,
		Gateways:  gateways,
		Ping:      pool.Ping,
		Log:       log,
		Options: httpapi.Options{
			AllowedOrigins: cfg.AllowedOrigins,
			TrustProxy:     cfg.TrustProxy,
			RateLimitRPS:   cfg.RateLimitRPS,
			RateLimitBurst: cfg.RateLimitBurst,
		},
	}

	srv := &http.Server{
		Addr:              ":" + cfg.HTTPPort,
		Handler:           api.Router(),
		ReadHeaderTimeout: 5 * time.Second,
		ReadTimeout:       15 * time.Second,
		WriteTimeout:      45 * time.Second, // checkout incluye cobro con reintentos
		IdleTimeout:       60 * time.Second,
	}
	errCh := make(chan error, 1)
	go func() {
		log.Info("servidor escuchando", "addr", srv.Addr, "env", cfg.Env)
		errCh <- srv.ListenAndServe()
	}()

	select {
	case err := <-errCh:
		if !errors.Is(err, http.ErrServerClosed) {
			return err
		}
	case <-ctx.Done():
		log.Info("apagando servidor")
		shutdownCtx, cancel := context.WithTimeout(context.Background(), 20*time.Second)
		defer cancel()
		return srv.Shutdown(shutdownCtx)
	}
	return nil
}

func checkoutCalculator(cfg config.Config, coupons pricing.CouponLookup) pricing.Calculator {
	return pricing.Chain(pricing.Base{Currency: cfg.Currency},
		pricing.WithShipping(cfg.ShippingCents, cfg.FreeShippingOverCents),
		pricing.WithCoupons(coupons),
	)
}
