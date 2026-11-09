package sales_test

// Pruebas de integracion contra PostgreSQL real. Se omiten si no hay base:
//
//	TEST_DATABASE_URL=postgres://ecommerce:ecommerce@localhost:5432/ecommerce?sslmode=disable go test ./internal/sales -race -count=1

import (
	"context"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"os"
	"sync"
	"testing"
	"time"

	"ecommerce/internal/apperr"
	"ecommerce/internal/auth"
	"ecommerce/internal/catalog"
	"ecommerce/internal/coupon"
	"ecommerce/internal/inventory"
	"ecommerce/internal/payment"
	"ecommerce/internal/platform/database"
	"ecommerce/internal/pricing"
	"ecommerce/internal/sales"
)

type env struct {
	ctx     context.Context
	cat     *catalog.Service
	inv     *inventory.Store
	sales   *sales.Service
	store   *sales.Store
	coupons *coupon.Store
	auth    *auth.Service
	suffix  string
}

func setup(t *testing.T) *env {
	t.Helper()
	url := os.Getenv("TEST_DATABASE_URL")
	if url == "" {
		t.Skip("TEST_DATABASE_URL no definido; se omite la prueba de integracion")
	}
	ctx := context.Background()
	log := slog.New(slog.NewTextHandler(io.Discard, nil))
	pool, err := database.Connect(ctx, url, log)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(pool.Close)
	if err := database.Migrate(ctx, pool, log); err != nil {
		t.Fatal(err)
	}

	inv := inventory.NewStore(pool)
	cat := catalog.NewService(catalog.NewRepository(pool), inv)
	coupons := coupon.NewStore(pool)
	calc := pricing.Chain(pricing.Base{Currency: "USD"},
		pricing.WithTax(10), pricing.WithShipping(500, 100000), pricing.WithCoupons(coupons))
	gws := payment.NewRegistry(payment.Decorate(payment.NewSimulated("stripe"), payment.WithRetry(3, time.Millisecond)))
	store := sales.NewStore(pool)
	return &env{
		ctx: ctx, cat: cat, inv: inv, coupons: coupons, store: store,
		sales:  sales.NewService(pool, store, cat, calc, gws, log),
		auth:   auth.NewService(pool, auth.NewTokenManager("test-secret-test-secret-test-secret-123", time.Hour)),
		suffix: fmt.Sprintf("%d", time.Now().UnixNano()),
	}
}

func (e *env) product(t *testing.T, name string, price int64, stock int) catalog.Record {
	t.Helper()
	rec, err := e.cat.CreateSimple(e.ctx, catalog.CreateSimpleInput{
		SKU: name + "-" + e.suffix, Name: name, PriceCents: price, InitialStock: stock,
	})
	if err != nil {
		t.Fatal(err)
	}
	return rec
}

func (e *env) stock(t *testing.T, id string) int {
	t.Helper()
	q, err := e.inv.Get(e.ctx, id)
	if err != nil {
		t.Fatal(err)
	}
	return q
}

func (e *env) user(t *testing.T, tag string) auth.User {
	t.Helper()
	u, err := e.auth.Register(e.ctx, auth.RegisterInput{
		Email: fmt.Sprintf("%s-%s@test.com", tag, e.suffix), Name: tag, Password: "password-segura-1",
	})
	if err != nil {
		t.Fatal(err)
	}
	return u
}

func params(u auth.User, token, key string, items ...sales.ItemInput) sales.CheckoutParams {
	return sales.CheckoutParams{
		UserID: u.ID, Email: u.Email, IdempotencyKey: key,
		Input: sales.CheckoutInput{
			Items: items, Gateway: "stripe", PaymentToken: token,
			Shipping: sales.Shipping{Name: "Ana", Address: "Calle 1", City: "Bogota", Country: "CO"},
		},
	}
}

func TestCheckoutWithBundleDeductsLeafStockAndRefundRestoresIt(t *testing.T) {
	e := setup(t)
	mouse := e.product(t, "mouse", 2000, 10)
	kb := e.product(t, "kb", 5000, 10)
	bundle, err := e.cat.CreateBundle(e.ctx, catalog.CreateBundleInput{
		SKU: "kit-" + e.suffix, Name: "Kit", DiscountPct: 10,
		Items: []catalog.BundleItem{{ComponentID: mouse.ID, Quantity: 1}, {ComponentID: kb.ID, Quantity: 2}},
	})
	if err != nil {
		t.Fatal(err)
	}
	u := e.user(t, "buyer1")

	order, err := e.sales.Checkout(e.ctx, params(u, "tok_ok", "", sales.ItemInput{ProductID: bundle.ID, Quantity: 2}))
	if err != nil {
		t.Fatal(err)
	}
	if order.Status != sales.StatusPaid || order.Payment == nil || order.Payment.Status != "approved" {
		t.Fatalf("orden inesperada: %+v", order)
	}
	// kit = (2000 + 2*5000) * 0.9 = 10800 ; x2 = 21600 ; envio 500 ; impuesto 10% de 21600 = 2160
	if order.SubtotalCents != 21600 || order.TaxCents != 2160 || order.TotalCents != 21600+500+2160 {
		t.Fatalf("totales inesperados: %+v", order)
	}
	if e.stock(t, mouse.ID) != 8 || e.stock(t, kb.ID) != 6 {
		t.Fatalf("stock de hojas incorrecto: mouse=%d kb=%d", e.stock(t, mouse.ID), e.stock(t, kb.ID))
	}

	refunded, err := e.sales.Refund(e.ctx, order.ID, "prueba", sales.StatusRefunded)
	if err != nil {
		t.Fatal(err)
	}
	if refunded.Status != sales.StatusRefunded || refunded.Payment.Status != "refunded" {
		t.Fatalf("reembolso inesperado: %+v", refunded)
	}
	if e.stock(t, mouse.ID) != 10 || e.stock(t, kb.ID) != 10 {
		t.Fatal("el reembolso debio reponer el inventario")
	}
	if _, err := e.sales.Refund(e.ctx, order.ID, "otra vez", sales.StatusRefunded); !errors.Is(err, apperr.ErrConflict) {
		t.Fatalf("un segundo reembolso debe ser conflicto, obtuvo %v", err)
	}
}

func TestDeclinedPaymentRestoresStockAndCoupon(t *testing.T) {
	e := setup(t)
	p := e.product(t, "lampara", 3000, 5)
	code := "T" + e.suffix[len(e.suffix)-8:]
	if _, err := e.coupons.Create(e.ctx, coupon.CreateInput{Code: code, DiscountPct: 10, MaxUses: 1}); err != nil {
		t.Fatal(err)
	}
	u := e.user(t, "buyer2")

	in := params(u, "tok_declined", "", sales.ItemInput{ProductID: p.ID, Quantity: 2})
	in.Input.Coupon = code
	order, err := e.sales.Checkout(e.ctx, in)
	if err != nil {
		t.Fatal(err)
	}
	if order.Status != sales.StatusFailed || order.Payment.Status != "declined" {
		t.Fatalf("debio quedar fallida: %+v", order)
	}
	if e.stock(t, p.ID) != 5 {
		t.Fatal("el pago rechazado debio devolver el stock")
	}
	// El cupon (max 1 uso) debe seguir disponible porque la venta fallo.
	if _, err := e.coupons.Percent(e.ctx, code); err != nil {
		t.Fatalf("el cupon debio liberarse: %v", err)
	}
	ok := params(u, "tok_ok", "", sales.ItemInput{ProductID: p.ID, Quantity: 1})
	ok.Input.Coupon = code
	if o, err := e.sales.Checkout(e.ctx, ok); err != nil || o.Status != sales.StatusPaid || o.DiscountCents != 300 {
		t.Fatalf("venta con cupon: %+v err=%v", o, err)
	}
	if _, err := e.coupons.Percent(e.ctx, code); err == nil {
		t.Fatal("el cupon de 1 uso debio agotarse")
	}
}

func TestInsufficientStockRejectsSale(t *testing.T) {
	e := setup(t)
	p := e.product(t, "raro", 1000, 1)
	u := e.user(t, "buyer3")
	_, err := e.sales.Checkout(e.ctx, params(u, "tok_ok", "", sales.ItemInput{ProductID: p.ID, Quantity: 2}))
	if !errors.Is(err, apperr.ErrInsufficientStock) {
		t.Fatalf("esperaba stock insuficiente, obtuvo %v", err)
	}
	if e.stock(t, p.ID) != 1 {
		t.Fatal("una venta rechazada no debe tocar el stock")
	}
}

func TestIdempotencyKeyReturnsSameOrder(t *testing.T) {
	e := setup(t)
	p := e.product(t, "reloj", 4000, 10)
	u := e.user(t, "buyer4")
	in := params(u, "tok_ok", "clave-"+e.suffix, sales.ItemInput{ProductID: p.ID, Quantity: 1})

	first, err := e.sales.Checkout(e.ctx, in)
	if err != nil {
		t.Fatal(err)
	}
	second, err := e.sales.Checkout(e.ctx, in)
	if err != nil {
		t.Fatal(err)
	}
	if first.ID != second.ID || e.stock(t, p.ID) != 9 {
		t.Fatalf("la repeticion debio devolver la misma orden sin descontar otra vez (stock=%d)", e.stock(t, p.ID))
	}
}

func TestConcurrentCheckoutsNeverOversell(t *testing.T) {
	e := setup(t)
	p := e.product(t, "ultimo", 1000, 3)

	const buyers = 10
	var wg sync.WaitGroup
	var mu sync.Mutex
	paid := 0
	for i := 0; i < buyers; i++ {
		u := e.user(t, fmt.Sprintf("conc%d", i))
		wg.Add(1)
		go func() {
			defer wg.Done()
			o, err := e.sales.Checkout(e.ctx, params(u, "tok_ok", "", sales.ItemInput{ProductID: p.ID, Quantity: 1}))
			if err == nil && o.Status == sales.StatusPaid {
				mu.Lock()
				paid++
				mu.Unlock()
			}
		}()
	}
	wg.Wait()
	if paid != 3 || e.stock(t, p.ID) != 0 {
		t.Fatalf("debian venderse exactamente 3 (vendidas=%d, stock=%d)", paid, e.stock(t, p.ID))
	}
}

func TestFulfillmentFlow(t *testing.T) {
	e := setup(t)
	p := e.product(t, "silla", 9000, 4)
	u := e.user(t, "buyer5")
	order, err := e.sales.Checkout(e.ctx, params(u, "tok_ok", "", sales.ItemInput{ProductID: p.ID, Quantity: 1}))
	if err != nil {
		t.Fatal(err)
	}
	if _, err := e.sales.MarkDelivered(e.ctx, order.ID); err == nil {
		t.Fatal("no se puede entregar una orden que no fue enviada")
	}
	if o, err := e.sales.MarkShipped(e.ctx, order.ID); err != nil || o.Status != sales.StatusShipped {
		t.Fatalf("envio: %+v %v", o, err)
	}
	if _, err := e.sales.Refund(e.ctx, order.ID, "x", sales.StatusRefunded); !errors.Is(err, apperr.ErrConflict) {
		t.Fatalf("una orden enviada no se reembolsa por esta via, obtuvo %v", err)
	}
	if o, err := e.sales.MarkDelivered(e.ctx, order.ID); err != nil || o.Status != sales.StatusDelivered {
		t.Fatalf("entrega: %+v %v", o, err)
	}
}

func TestAuthRegisterLogin(t *testing.T) {
	e := setup(t)
	u := e.user(t, "login1")
	if _, err := e.auth.Register(e.ctx, auth.RegisterInput{Email: u.Email, Name: "x", Password: "password-segura-1"}); !errors.Is(err, apperr.ErrConflict) {
		t.Fatalf("email duplicado debe ser conflicto, obtuvo %v", err)
	}
	res, err := e.auth.Login(e.ctx, u.Email, "password-segura-1")
	if err != nil {
		t.Fatal(err)
	}
	p, err := e.auth.Authenticate(e.ctx, res.Token)
	if err != nil || p.UserID != u.ID || p.Role != auth.RoleCustomer {
		t.Fatalf("principal inesperado: %+v %v", p, err)
	}
	if _, err := e.auth.Login(e.ctx, u.Email, "incorrecta"); !errors.Is(err, apperr.ErrUnauthorized) {
		t.Fatalf("clave incorrecta debe ser 401, obtuvo %v", err)
	}
}
