package sales

// Capa de datos del checkout: escrituras transaccionales de ordenes, pagos,
// cupones e inventario. El servicio decide QUE hacer; aqui vive el SQL.

import (
	"context"
	"sort"
	"strings"

	"ecommerce/internal/apperr"
	"ecommerce/internal/inventory"
	"ecommerce/internal/platform/database"
	"ecommerce/internal/pricing"

	"github.com/jackc/pgx/v5"
)

// pendingOrder es todo lo necesario para registrar una orden antes de cobrarla.
type pendingOrder struct {
	Params CheckoutParams
	Items  []pricing.Item
	Needs  map[string]int // producto simple -> unidades a descontar
	Quote  pricing.Breakdown
}

// paymentRecord es un renglon de la tabla payments.
type paymentRecord struct {
	OrderID      string
	Gateway      string
	Status       string
	AmountCents  int64
	ExternalRef  string
	ErrorMessage string
}

// inTx ejecuta fn dentro de una transaccion: confirma si no hay error y
// revierte en cualquier otro caso.
func (s *Store) inTx(ctx context.Context, fn func(pgx.Tx) error) error {
	tx, err := s.pool.Begin(ctx)
	if err != nil {
		return err
	}
	defer tx.Rollback(ctx) //nolint:errcheck
	if err := fn(tx); err != nil {
		return database.MapError(err)
	}
	return database.MapError(tx.Commit(ctx))
}

// createPending crea la orden "pending", consume el cupon y descuenta el
// inventario en una sola transaccion. Devuelve el id de la orden.
func (s *Store) createPending(ctx context.Context, o pendingOrder) (string, error) {
	var orderID string
	err := s.inTx(ctx, func(tx pgx.Tx) error {
		var err error
		if orderID, err = insertOrder(ctx, tx, o); err != nil {
			return err
		}
		if err := insertItems(ctx, tx, orderID, o.Items); err != nil {
			return err
		}
		if err := consumeCoupon(ctx, tx, o.couponCode()); err != nil {
			return err
		}
		return reserveStock(ctx, tx, orderID, o.Needs)
	})
	return orderID, err
}

func (o pendingOrder) couponCode() string {
	return strings.ToUpper(strings.TrimSpace(o.Params.Input.Coupon))
}

func insertOrder(ctx context.Context, tx pgx.Tx, o pendingOrder) (string, error) {
	p, q, sh := o.Params, o.Quote, o.Params.Input.Shipping
	var orderID string
	err := tx.QueryRow(ctx,
		`INSERT INTO orders (user_id, idempotency_key, customer_email, status, coupon, subtotal_cents, discount_cents,
			shipping_cents, tax_cents, total_cents, currency, ship_name, ship_address, ship_city, ship_country, ship_phone)
		 VALUES (NULLIF($1::text, '')::uuid, NULLIF($2, ''), $3, 'pending', $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14, $15)
		 RETURNING id::text`,
		p.UserID, p.IdempotencyKey, p.Email, o.couponCode(), q.SubtotalCents, q.DiscountCents, q.ShippingCents, q.TaxCents,
		q.TotalCents, q.Currency, sh.Name, sh.Address, sh.City, sh.Country, sh.Phone).Scan(&orderID)
	return orderID, database.MapError(err)
}

func insertItems(ctx context.Context, tx pgx.Tx, orderID string, items []pricing.Item) error {
	for _, it := range items {
		if _, err := tx.Exec(ctx,
			`INSERT INTO order_items (order_id, product_id, name, unit_price_cents, quantity) VALUES ($1::uuid, $2::uuid, $3, $4, $5)`,
			orderID, it.ProductID, it.Name, it.UnitPriceCents, it.Quantity); err != nil {
			return err
		}
	}
	return nil
}

// consumeCoupon gasta un uso del cupon solo si sigue vigente y con cupo.
func consumeCoupon(ctx context.Context, tx pgx.Tx, code string) error {
	if code == "" {
		return nil
	}
	tag, err := tx.Exec(ctx,
		`UPDATE coupons SET used_count = used_count + 1
		 WHERE code = $1 AND active AND (max_uses = 0 OR used_count < max_uses)
		   AND (expires_at IS NULL OR expires_at > now())`, code)
	if err != nil {
		return err
	}
	if tag.RowsAffected() == 0 {
		return apperr.Validation("el cupon %s ya no esta disponible", code)
	}
	return nil
}

// releaseCoupon devuelve el uso del cupon de una orden (si tenia).
func releaseCoupon(ctx context.Context, tx pgx.Tx, orderID string) error {
	_, err := tx.Exec(ctx,
		`UPDATE coupons SET used_count = GREATEST(used_count - 1, 0)
		 WHERE code = (SELECT NULLIF(coupon, '') FROM orders WHERE id = $1::uuid)`, orderID)
	return err
}

// reserveStock descuenta inventario y anota que se desconto para la orden.
// Bloquea en orden fijo para evitar deadlocks entre ventas concurrentes.
func reserveStock(ctx context.Context, tx pgx.Tx, orderID string, needs map[string]int) error {
	for _, id := range sortedKeys(needs) {
		move := inventory.StockMove{ProductID: id, Quantity: needs[id], OrderRef: orderID}
		if err := inventory.Reserve(ctx, tx, move); err != nil {
			return err
		}
		if _, err := tx.Exec(ctx,
			`INSERT INTO order_stock_items (order_id, product_id, quantity) VALUES ($1::uuid, $2::uuid, $3)`,
			orderID, id, needs[id]); err != nil {
			return err
		}
	}
	return nil
}

func releaseStock(ctx context.Context, tx pgx.Tx, orderID string, needs map[string]int) error {
	for _, id := range sortedKeys(needs) {
		move := inventory.StockMove{ProductID: id, Quantity: needs[id], OrderRef: orderID}
		if err := inventory.Release(ctx, tx, move); err != nil {
			return err
		}
	}
	return nil
}

func setStatus(ctx context.Context, tx pgx.Tx, orderID, status string) error {
	_, err := tx.Exec(ctx, `UPDATE orders SET status = $2, updated_at = now() WHERE id = $1::uuid`, orderID, status)
	return err
}

func insertPayment(ctx context.Context, tx pgx.Tx, p paymentRecord) error {
	_, err := tx.Exec(ctx,
		`INSERT INTO payments (order_id, gateway, status, amount_cents, external_ref, error_message)
		 VALUES ($1::uuid, $2, $3, $4, $5, $6)`,
		p.OrderID, p.Gateway, p.Status, p.AmountCents, p.ExternalRef, p.ErrorMessage)
	return err
}

// markPaid deja la orden pagada y registra el cobro aprobado.
func (s *Store) markPaid(ctx context.Context, p paymentRecord) error {
	return s.inTx(ctx, func(tx pgx.Tx) error {
		if err := setStatus(ctx, tx, p.OrderID, StatusPaid); err != nil {
			return err
		}
		return insertPayment(ctx, tx, p)
	})
}

// markFailed compensa una venta cuyo cobro fallo: devuelve stock y cupon,
// deja la orden fallida y registra el intento de pago.
func (s *Store) markFailed(ctx context.Context, p paymentRecord, needs map[string]int) error {
	return s.inTx(ctx, func(tx pgx.Tx) error {
		if err := releaseStock(ctx, tx, p.OrderID, needs); err != nil {
			return err
		}
		if err := releaseCoupon(ctx, tx, p.OrderID); err != nil {
			return err
		}
		if err := setStatus(ctx, tx, p.OrderID, StatusFailed); err != nil {
			return err
		}
		return insertPayment(ctx, tx, p)
	})
}

func sortedKeys(m map[string]int) []string {
	keys := make([]string, 0, len(m))
	for k := range m {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	return keys
}
