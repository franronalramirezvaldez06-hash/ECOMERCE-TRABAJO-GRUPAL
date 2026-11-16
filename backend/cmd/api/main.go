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

	"github.com/jackc/pgx/v5/pgxpool"
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
	if !cfg.IsProduction() {
		log.Warn("modo desarrollo: no uses esta configuracion en produccion", "env", cfg.Env)
	}
	api, err := newAPI(ctx, app{cfg: cfg, pool: pool, log: log})
	if err != nil {
		return err
	}
	return serve(ctx, newServer(cfg, api.Router()), log)
}

// app agrupa la infraestructura ya inicializada que necesitan los modulos.
type app struct {
	cfg  config.Config
	pool *pgxpool.Pool
	log  *slog.Logger
}

// newAPI es la raiz de composicion: crea cada modulo y le inyecta sus dependencias.
func newAPI(ctx context.Context, a app) (*httpapi.API, error) {
	cfg, pool, log := a.cfg, a.pool, a.log
	authSvc := auth.NewService(auth.NewRepository(pool), auth.NewTokenManager(cfg.JWTSecret, cfg.JWTTTL))
	if err := ensureAdmin(ctx, authSvc, a); err != nil {
		return nil, err
	}
	coupons := coupon.NewStore(pool)
	gateways := newGateways(cfg, log)
	inv := inventory.NewStore(pool)
	cat := catalog.NewService(catalog.NewRepository(pool), inv)
	orders := sales.NewStore(pool)
	salesSvc := sales.NewService(sales.Deps{
		Store: orders, Catalog: cat, Calculator: checkoutCalculator(cfg, coupons), Gateways: gateways, Log: log,
	})
	return &httpapi.API{
		Auth:      authSvc,
		Catalog:   cat,
		Inventory: inv,
		Coupons:   coupons,
		Sales:     salesSvc,
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
	}, nil
}

func ensureAdmin(ctx context.Context, authSvc *auth.Service, a app) error {
	if a.cfg.AdminEmail == "" {
		return nil
	}
	created, err := authSvc.EnsureAdmin(ctx, a.cfg.AdminEmail, a.cfg.AdminPassword)
	if created {
		a.log.Info("administrador inicial creado", "email", a.cfg.AdminEmail)
	}
	return err
}

// newGateways registra las pasarelas disponibles, cada una envuelta con los
// decoradores: logging -> reintentos -> timeout -> limite de monto.
func newGateways(cfg config.Config, log *slog.Logger) *payment.Registry {
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
	return payment.NewRegistry(gws...)
}

func newServer(cfg config.Config, handler http.Handler) *http.Server {
	return &http.Server{
		Addr:              ":" + cfg.HTTPPort,
		Handler:           handler,
		ReadHeaderTimeout: 5 * time.Second,
		ReadTimeout:       15 * time.Second,
		WriteTimeout:      45 * time.Second, // checkout incluye cobro con reintentos
		IdleTimeout:       60 * time.Second,
	}
}

// serve atiende peticiones hasta que el servidor falle o llegue una senal de
// apagado; en ese caso espera a que terminen las peticiones en curso.
func serve(ctx context.Context, srv *http.Server, log *slog.Logger) error {
	errCh := make(chan error, 1)
	go func() {
		log.Info("servidor escuchando", "addr", srv.Addr)
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

// Decorator (precios): subtotal -> cupon -> envio.
func checkoutCalculator(cfg config.Config, coupons pricing.CouponLookup) pricing.Calculator {
	return pricing.Chain(pricing.Base{Currency: cfg.Currency},
		pricing.WithShipping(cfg.ShippingCents, cfg.FreeShippingOverCents),
		pricing.WithCoupons(coupons),
	)
}
