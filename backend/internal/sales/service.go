package sales

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"sort"
	"strings"

	"ecommerce/internal/apperr"
	"ecommerce/internal/catalog"
	"ecommerce/internal/inventory"
	"ecommerce/internal/payment"
	"ecommerce/internal/platform/database"
	"ecommerce/internal/pricing"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

// Catalog es lo que necesita ventas del modulo de catalogo.
type Catalog interface {
	Build(ctx context.Context, id string) (catalog.Component, error)
}

type Service struct {
	pool       *pgxpool.Pool
	store      *Store
	catalog    Catalog
	calculator pricing.Calculator
	gateways   *payment.Registry
	log        *slog.Logger
}

func NewService(pool *pgxpool.Pool, store *Store, c Catalog, calc pricing.Calculator, gw *payment.Registry, log *slog.Logger) *Service {
	return &Service{pool: pool, store: store, catalog: c, calculator: calc, gateways: gw, log: log}
}

// cart es el resultado de resolver los items contra el catalogo (Composite).
type cart struct {
	items []pricing.Item
	needs map[string]int // producto simple -> unidades a descontar
}

// resolve arma el carrito: precio unitario por componente y requerimientos de
// inventario ya expandidos (los combos se convierten en productos simples).
func (s *Service) resolve(ctx context.Context, inputs []ItemInput) (*cart, error) {
	if err := validateItems(inputs); err != nil {
		return nil, err
	}
	merged := map[string]int{}
	var order []string
	for _, in := range inputs {
		id := strings.TrimSpace(in.ProductID)
		if _, ok := merged[id]; !ok {
			order = append(order, id)
		}
		merged[id] += in.Quantity
	}
	c := &cart{needs: map[string]int{}}
	for _, id := range order {
		comp, err := s.catalog.Build(ctx, id)
		if err != nil {
			return nil, err
		}
		qty := merged[id]
		c.items = append(c.items, pricing.Item{ProductID: id, Name: comp.Name(), UnitPriceCents: comp.Price(), Quantity: qty})
		for leaf, n := range comp.Requirements(qty) {
			c.needs[leaf] += n
		}
	}
	return c, nil
}

// Quote calcula el desglose de precios sin reservar nada.
func (s *Service) Quote(ctx context.Context, in QuoteInput) (pricing.Breakdown, error) {
	c, err := s.resolve(ctx, in.Items)
	if err != nil {
		return pricing.Breakdown{}, err
	}
	return s.calculator.Calculate(ctx, pricing.Request{Items: c.items, Coupon: in.Coupon})
}

// Checkout ejecuta la venta:
//  1. si la clave de idempotencia ya existe, devuelve la orden anterior
//  2. resuelve carrito y precio
//  3. en una transaccion crea la orden, consume el cupon y descuenta inventario
//  4. cobra en la pasarela (fuera de la transaccion)
//  5. si el cobro falla, compensa: devuelve stock y cupon y marca la orden fallida
//
// Un pago rechazado NO es un error: devuelve la orden con status "failed".
func (s *Service) Checkout(ctx context.Context, p CheckoutParams) (*Order, error) {
	in := p.Input
	if err := in.validate(); err != nil {
		return nil, err
	}
	if p.IdempotencyKey != "" {
		id, err := s.store.idByIdempotencyKey(ctx, p.UserID, p.IdempotencyKey)
		if err == nil {
			return s.store.Get(ctx, id)
		}
		if !errors.Is(err, apperr.ErrNotFound) {
			return nil, err
		}
	}
	gw, err := s.gateways.Get(in.Gateway)
	if err != nil {
		return nil, err
	}
	c, err := s.resolve(ctx, in.Items)
	if err != nil {
		return nil, err
	}
	quote, err := s.calculator.Calculate(ctx, pricing.Request{Items: c.items, Coupon: in.Coupon})
	if err != nil {
		return nil, err
	}

	orderID, err := s.createPending(ctx, p, c, quote)
	if err != nil {
		// Peticion duplicada concurrente con la misma clave: devolver la ganadora.
		if p.IdempotencyKey != "" && errors.Is(err, apperr.ErrConflict) {
			if id, e := s.store.idByIdempotencyKey(ctx, p.UserID, p.IdempotencyKey); e == nil {
				return s.store.Get(ctx, id)
			}
		}
		return nil, err
	}

	res, chargeErr := gw.Charge(ctx, payment.ChargeRequest{
		Reference: orderID, AmountCents: quote.TotalCents, Currency: quote.Currency,
		Token: in.PaymentToken, Email: p.Email,
	})

	// Las escrituras posteriores no deben cancelarse si el cliente se desconecta.
	bg := context.WithoutCancel(ctx)
	if chargeErr != nil {
		if err := s.markFailed(bg, orderID, gw.Name(), quote.TotalCents, c.needs, chargeErr); err != nil {
			s.log.Error("no se pudo compensar una venta fallida", "order", orderID, "error", err)
			return nil, err
		}
	} else if err := s.markPaid(bg, orderID, gw.Name(), quote.TotalCents, res.ExternalRef); err != nil {
		// Cobro hecho pero sin registrar: intentar devolver el dinero.
		s.log.Error("cobro aprobado pero no registrado, intentando reembolso", "order", orderID, "error", err)
		if refundErr := gw.Refund(bg, res.ExternalRef, quote.TotalCents); refundErr != nil {
			s.log.Error("REEMBOLSO FALLIDO, requiere revision manual", "order", orderID, "external_ref", res.ExternalRef, "error", refundErr)
		}
		return nil, err
	}
	return s.store.Get(bg, orderID)
}

func (s *Service) createPending(ctx context.Context, p CheckoutParams, c *cart, q pricing.Breakdown) (string, error) {
	tx, err := s.pool.Begin(ctx)
	if err != nil {
		return "", err
	}
	defer tx.Rollback(ctx) //nolint:errcheck

	coupon := strings.ToUpper(strings.TrimSpace(p.Input.Coupon))
	sh := p.Input.Shipping
	var orderID string
	err = tx.QueryRow(ctx,
		`INSERT INTO orders (user_id, idempotency_key, customer_email, status, coupon, subtotal_cents, discount_cents,
			shipping_cents, tax_cents, total_cents, currency, ship_name, ship_address, ship_city, ship_country, ship_phone)
		 VALUES (NULLIF($1::text, '')::uuid, NULLIF($2, ''), $3, 'pending', $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14, $15)
		 RETURNING id::text`,
		p.UserID, p.IdempotencyKey, p.Email, coupon, q.SubtotalCents, q.DiscountCents, q.ShippingCents, q.TaxCents,
		q.TotalCents, q.Currency, sh.Name, sh.Address, sh.City, sh.Country, sh.Phone).Scan(&orderID)
	if err != nil {
		return "", database.MapError(err)
	}
	for _, it := range c.items {
		if _, err := tx.Exec(ctx,
			`INSERT INTO order_items (order_id, product_id, name, unit_price_cents, quantity) VALUES ($1::uuid, $2::uuid, $3, $4, $5)`,
			orderID, it.ProductID, it.Name, it.UnitPriceCents, it.Quantity); err != nil {
			return "", database.MapError(err)
		}
	}
	if coupon != "" {
		tag, err := tx.Exec(ctx,
			`UPDATE coupons SET used_count = used_count + 1
			 WHERE code = $1 AND active AND (max_uses = 0 OR used_count < max_uses)
			   AND (expires_at IS NULL OR expires_at > now())`, coupon)
		if err != nil {
			return "", database.MapError(err)
		}
		if tag.RowsAffected() == 0 {
			return "", apperr.Validation("el cupon %s ya no esta disponible", coupon)
		}
	}
	// Orden fija de bloqueo para evitar deadlocks entre ventas concurrentes.
	for _, id := range sortedKeys(c.needs) {
		if err := inventory.Reserve(ctx, tx, id, c.needs[id], orderID); err != nil {
			return "", err
		}
		if _, err := tx.Exec(ctx,
			`INSERT INTO order_stock_items (order_id, product_id, quantity) VALUES ($1::uuid, $2::uuid, $3)`,
			orderID, id, c.needs[id]); err != nil {
			return "", database.MapError(err)
		}
	}
	if err := tx.Commit(ctx); err != nil {
		return "", database.MapError(err)
	}
	return orderID, nil
}

func (s *Service) markPaid(ctx context.Context, orderID, gateway string, amount int64, externalRef string) error {
	return s.inTx(ctx, func(tx pgx.Tx) error {
		if _, err := tx.Exec(ctx, `UPDATE orders SET status = 'paid', updated_at = now() WHERE id = $1::uuid`, orderID); err != nil {
			return err
		}
		_, err := tx.Exec(ctx,
			`INSERT INTO payments (order_id, gateway, status, amount_cents, external_ref) VALUES ($1::uuid, $2, 'approved', $3, $4)`,
			orderID, gateway, amount, externalRef)
		return err
	})
}

func (s *Service) markFailed(ctx context.Context, orderID, gateway string, amount int64, needs map[string]int, cause error) error {
	status := "error"
	if errors.Is(cause, payment.ErrDeclined) {
		status = "declined"
	}
	return s.inTx(ctx, func(tx pgx.Tx) error {
		for _, id := range sortedKeys(needs) {
			if err := inventory.Release(ctx, tx, id, needs[id], orderID); err != nil {
				return err
			}
		}
		if _, err := tx.Exec(ctx,
			`UPDATE coupons SET used_count = GREATEST(used_count - 1, 0)
			 WHERE code = (SELECT NULLIF(coupon, '') FROM orders WHERE id = $1::uuid)`, orderID); err != nil {
			return err
		}
		if _, err := tx.Exec(ctx, `UPDATE orders SET status = 'failed', updated_at = now() WHERE id = $1::uuid`, orderID); err != nil {
			return err
		}
		_, err := tx.Exec(ctx,
			`INSERT INTO payments (order_id, gateway, status, amount_cents, error_message) VALUES ($1::uuid, $2, $3, $4, $5)`,
			orderID, gateway, status, amount, cause.Error())
		return err
	})
}

// Refund devuelve el dinero de una orden pagada (y aun no enviada), repone el
// inventario y deja la orden en finalStatus ("refunded" o "cancelled").
// Es atomico: el UPDATE condicional impide reembolsar dos veces y, si la
// pasarela falla, la transaccion se revierte y la orden sigue "paid".
func (s *Service) Refund(ctx context.Context, orderID, reason, finalStatus string) (*Order, error) {
	if finalStatus != StatusRefunded && finalStatus != StatusCancelled {
		return nil, apperr.Validation("estado final invalido")
	}
	tx, err := s.pool.Begin(ctx)
	if err != nil {
		return nil, err
	}
	defer tx.Rollback(ctx) //nolint:errcheck

	var total int64
	err = tx.QueryRow(ctx,
		`UPDATE orders SET status = $2, updated_at = now() WHERE id = $1::uuid AND status = 'paid' RETURNING total_cents`,
		orderID, finalStatus).Scan(&total)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			if _, getErr := s.store.Get(ctx, orderID); getErr != nil {
				return nil, getErr
			}
			return nil, fmt.Errorf("%w: solo se pueden reembolsar o cancelar ordenes en estado paid", apperr.ErrConflict)
		}
		return nil, database.MapError(err)
	}

	var gwName, externalRef string
	if err := tx.QueryRow(ctx,
		`SELECT gateway, external_ref FROM payments WHERE order_id = $1::uuid AND status = 'approved'
		 ORDER BY created_at DESC LIMIT 1`, orderID).Scan(&gwName, &externalRef); err != nil {
		return nil, database.MapError(err)
	}
	gw, err := s.gateways.Get(gwName)
	if err != nil {
		return nil, fmt.Errorf("%w: pasarela %q no disponible para reembolsar", apperr.ErrGateway, gwName)
	}
	if err := gw.Refund(ctx, externalRef, total); err != nil {
		s.log.Error("la pasarela rechazo el reembolso", "order", orderID, "gateway", gwName, "error", err)
		return nil, fmt.Errorf("%w: %v", apperr.ErrGateway, err)
	}
	if _, err := tx.Exec(ctx,
		`INSERT INTO payments (order_id, gateway, status, amount_cents, external_ref, error_message)
		 VALUES ($1::uuid, $2, 'refunded', $3, $4, $5)`, orderID, gwName, total, externalRef, reason); err != nil {
		return nil, database.MapError(err)
	}

	rows, err := tx.Query(ctx,
		`SELECT product_id::text, quantity FROM order_stock_items WHERE order_id = $1::uuid ORDER BY product_id`, orderID)
	if err != nil {
		return nil, err
	}
	needs := map[string]int{}
	for rows.Next() {
		var id string
		var q int
		if err := rows.Scan(&id, &q); err != nil {
			rows.Close()
			return nil, err
		}
		needs[id] = q
	}
	rows.Close()
	if err := rows.Err(); err != nil {
		return nil, err
	}
	for _, id := range sortedKeys(needs) {
		if err := inventory.Release(ctx, tx, id, needs[id], orderID); err != nil {
			return nil, err
		}
	}
	if err := tx.Commit(ctx); err != nil {
		return nil, database.MapError(err)
	}
	return s.store.Get(ctx, orderID)
}

// MarkShipped y MarkDelivered avanzan el cumplimiento de una orden pagada.
func (s *Service) MarkShipped(ctx context.Context, orderID string) (*Order, error) {
	if err := s.store.UpdateStatus(ctx, orderID, StatusPaid, StatusShipped); err != nil {
		return nil, err
	}
	return s.store.Get(ctx, orderID)
}

func (s *Service) MarkDelivered(ctx context.Context, orderID string) (*Order, error) {
	if err := s.store.UpdateStatus(ctx, orderID, StatusShipped, StatusDelivered); err != nil {
		return nil, err
	}
	return s.store.Get(ctx, orderID)
}

func (s *Service) inTx(ctx context.Context, fn func(pgx.Tx) error) error {
	tx, err := s.pool.Begin(ctx)
	if err != nil {
		return err
	}
	defer tx.Rollback(ctx) //nolint:errcheck
	if err := fn(tx); err != nil {
		return fmt.Errorf("transaccion de venta: %w", database.MapError(err))
	}
	return tx.Commit(ctx)
}

func sortedKeys(m map[string]int) []string {
	keys := make([]string, 0, len(m))
	for k := range m {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	return keys
}
