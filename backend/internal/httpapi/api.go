// Package httpapi expone el backend como API REST JSON.
package httpapi

import (
	"context"
	"log/slog"
	"net/http"
	"time"

	"ecommerce/internal/auth"
	"ecommerce/internal/catalog"
	"ecommerce/internal/coupon"
	"ecommerce/internal/inventory"
	"ecommerce/internal/payment"
	"ecommerce/internal/sales"
)

type Options struct {
	AllowedOrigins []string
	TrustProxy     bool
	RateLimitRPS   float64
	RateLimitBurst int
}

type API struct {
	Auth      *auth.Service
	Catalog   *catalog.Service
	Inventory *inventory.Store
	Coupons   *coupon.Store
	Sales     *sales.Service
	Orders    *sales.Store
	Gateways  *payment.Registry
	Ping      func(context.Context) error
	Log       *slog.Logger
	Options   Options
}

func (a *API) Router() http.Handler {
	mux := http.NewServeMux()
	mux.HandleFunc("GET /health", a.health)
	a.authRoutes(mux)
	a.catalogRoutes(mux)
	a.inventoryRoutes(mux)
	a.couponRoutes(mux)
	a.salesRoutes(mux)
	return a.withMiddleware(mux)
}

func (a *API) authRoutes(mux *http.ServeMux) {
	// Login y registro: limite estricto contra fuerza bruta (1 intento/s, rafaga 5).
	strict := newLimiter(1, 5)
	mux.Handle("POST /api/v1/auth/register", strict.wrap(a.TrustProxy(), http.HandlerFunc(a.register)))
	mux.Handle("POST /api/v1/auth/login", strict.wrap(a.TrustProxy(), http.HandlerFunc(a.login)))
	mux.HandleFunc("GET /api/v1/auth/me", a.requireAuth(a.me))
}

// Catalogo: lectura publica, escritura solo administradores.
func (a *API) catalogRoutes(mux *http.ServeMux) {
	admin := a.requireAdmin
	mux.HandleFunc("GET /api/v1/products", a.listProducts)
	mux.HandleFunc("GET /api/v1/products/{id}", a.getProduct)
	mux.HandleFunc("POST /api/v1/products", admin(a.createProduct))
	mux.HandleFunc("PUT /api/v1/products/{id}", admin(a.updateProduct))
	mux.HandleFunc("POST /api/v1/bundles", admin(a.createBundle))
	mux.HandleFunc("GET /api/v1/categories", a.listCategories)
	mux.HandleFunc("POST /api/v1/categories", admin(a.createCategory))
}

func (a *API) inventoryRoutes(mux *http.ServeMux) {
	admin := a.requireAdmin
	mux.HandleFunc("GET /api/v1/inventory/low-stock", admin(a.lowStock))
	mux.HandleFunc("GET /api/v1/inventory/{id}", admin(a.getStock))
	mux.HandleFunc("PUT /api/v1/inventory/{id}", admin(a.setStock))
	mux.HandleFunc("POST /api/v1/inventory/{id}/adjust", admin(a.adjustStock))
	mux.HandleFunc("GET /api/v1/inventory/{id}/movements", admin(a.stockMovements))
}

func (a *API) couponRoutes(mux *http.ServeMux) {
	admin := a.requireAdmin
	mux.HandleFunc("GET /api/v1/coupons", admin(a.listCoupons))
	mux.HandleFunc("POST /api/v1/coupons", admin(a.createCoupon))
	mux.HandleFunc("PATCH /api/v1/coupons/{code}", admin(a.setCouponActive))
}

func (a *API) salesRoutes(mux *http.ServeMux) {
	admin, user := a.requireAdmin, a.requireAuth
	mux.HandleFunc("POST /api/v1/quotes", a.quote)
	mux.HandleFunc("POST /api/v1/orders", user(a.checkout))
	mux.HandleFunc("GET /api/v1/orders", user(a.listOrders))
	mux.HandleFunc("GET /api/v1/orders/{id}", user(a.getOrder))
	mux.HandleFunc("POST /api/v1/orders/{id}/cancel", user(a.cancelOrder))
	mux.HandleFunc("POST /api/v1/orders/{id}/refund", admin(a.refundOrder))
	mux.HandleFunc("POST /api/v1/orders/{id}/approve", admin(a.approveOrder))
	mux.HandleFunc("POST /api/v1/orders/{id}/ship", admin(a.shipOrder))
	mux.HandleFunc("POST /api/v1/orders/{id}/deliver", admin(a.deliverOrder))
	mux.HandleFunc("GET /api/v1/payments/gateways", a.listGateways)
	mux.HandleFunc("GET /api/v1/reports/sales", admin(a.salesReport))
}

// withMiddleware envuelve las rutas; el ultimo en envolver es el primero en ejecutarse.
func (a *API) withMiddleware(h http.Handler) http.Handler {
	general := newLimiter(a.Options.RateLimitRPS, a.Options.RateLimitBurst)
	h = a.authenticate(h)
	h = general.wrap(a.TrustProxy(), h)
	h = a.cors(h)
	h = a.logRequests(h)
	h = a.recoverer(h)
	h = securityHeaders(h)
	h = requestID(h)
	return h
}

// TrustProxy indica si se debe confiar en X-Forwarded-For para la IP del cliente.
func (a *API) TrustProxy() bool { return a.Options.TrustProxy }

func (a *API) health(w http.ResponseWriter, r *http.Request) {
	ctx, cancel := context.WithTimeout(r.Context(), 2*time.Second)
	defer cancel()
	if err := a.Ping(ctx); err != nil {
		writeJSON(w, http.StatusServiceUnavailable, map[string]string{"status": "db_down"})
		return
	}
	writeJSON(w, http.StatusOK, map[string]string{"status": "ok"})
}
