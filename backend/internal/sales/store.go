package sales

import (
	"context"
	"errors"
	"fmt"
	"time"

	"ecommerce/internal/apperr"
	"ecommerce/internal/platform/database"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

const orderColumns = `id::text, COALESCE(user_id::text, ''), customer_email, status, coupon, subtotal_cents,
	discount_cents, shipping_cents, tax_cents, total_cents, currency,
	ship_name, ship_address, ship_city, ship_country, ship_phone, created_at`

type Store struct{ pool *pgxpool.Pool }

func NewStore(pool *pgxpool.Pool) *Store { return &Store{pool: pool} }

func scanOrder(row pgx.Row) (Order, error) {
	var o Order
	err := row.Scan(&o.ID, &o.UserID, &o.CustomerEmail, &o.Status, &o.Coupon, &o.SubtotalCents,
		&o.DiscountCents, &o.ShippingCents, &o.TaxCents, &o.TotalCents, &o.Currency,
		&o.Shipping.Name, &o.Shipping.Address, &o.Shipping.City, &o.Shipping.Country, &o.Shipping.Phone, &o.CreatedAt)
	return o, database.MapError(err)
}

// Get devuelve la orden con sus items y su ultimo pago.
func (s *Store) Get(ctx context.Context, id string) (*Order, error) {
	o, err := scanOrder(s.pool.QueryRow(ctx, `SELECT `+orderColumns+` FROM orders WHERE id = $1::uuid`, id))
	if err != nil {
		return nil, err
	}
	rows, err := s.pool.Query(ctx,
		`SELECT product_id::text, name, unit_price_cents, quantity FROM order_items WHERE order_id = $1::uuid ORDER BY id`, id)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	for rows.Next() {
		var it OrderItem
		if err := rows.Scan(&it.ProductID, &it.Name, &it.UnitPriceCents, &it.Quantity); err != nil {
			return nil, err
		}
		o.Items = append(o.Items, it)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	rows.Close()

	var p Payment
	err = s.pool.QueryRow(ctx,
		`SELECT id::text, gateway, status, amount_cents, external_ref, error_message, created_at
		 FROM payments WHERE order_id = $1::uuid ORDER BY created_at DESC, id DESC LIMIT 1`, id).
		Scan(&p.ID, &p.Gateway, &p.Status, &p.AmountCents, &p.ExternalRef, &p.ErrorMessage, &p.CreatedAt)
	switch {
	case err == nil:
		o.Payment = &p
	case !errors.Is(err, pgx.ErrNoRows):
		return nil, err
	}
	return &o, nil
}

// List devuelve ordenes (sin items), de la mas reciente a la mas antigua.
// userID vacio = todas las ordenes (administrador).
func (s *Store) List(ctx context.Context, userID, status string, limit, offset int) ([]Order, error) {
	rows, err := s.pool.Query(ctx,
		`SELECT `+orderColumns+` FROM orders
		 WHERE ($1 = '' OR user_id::text = $1) AND ($2 = '' OR status = $2)
		 ORDER BY created_at DESC LIMIT $3 OFFSET $4`, userID, status, limit, offset)
	if err != nil {
		return nil, database.MapError(err)
	}
	defer rows.Close()
	out := []Order{}
	for rows.Next() {
		o, err := scanOrder(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, o)
	}
	return out, rows.Err()
}

// idByIdempotencyKey busca una orden previa creada con la misma clave.
func (s *Store) idByIdempotencyKey(ctx context.Context, userID, key string) (string, error) {
	var id string
	err := s.pool.QueryRow(ctx,
		`SELECT id::text FROM orders WHERE user_id = $1::uuid AND idempotency_key = $2`, userID, key).Scan(&id)
	return id, database.MapError(err)
}

// UpdateStatus avanza una orden de from a to de forma atomica.
func (s *Store) UpdateStatus(ctx context.Context, id, from, to string) error {
	tag, err := s.pool.Exec(ctx,
		`UPDATE orders SET status = $3, updated_at = now() WHERE id = $1::uuid AND status = $2`, id, from, to)
	if err != nil {
		return database.MapError(err)
	}
	if tag.RowsAffected() == 0 {
		if _, err := s.Get(ctx, id); err != nil {
			return err
		}
		return fmt.Errorf("%w: la orden no esta en estado %q", apperr.ErrConflict, from)
	}
	return nil
}

// ------------------------------------------------------------- reportes

type DayTotal struct {
	Day          string `json:"day"`
	Orders       int    `json:"orders"`
	RevenueCents int64  `json:"revenue_cents"`
}

type TopProduct struct {
	ProductID    string `json:"product_id"`
	Name         string `json:"name"`
	Units        int    `json:"units"`
	RevenueCents int64  `json:"revenue_cents"`
}

type Summary struct {
	From          time.Time      `json:"from"`
	To            time.Time      `json:"to"`
	OrdersByState map[string]int `json:"orders_by_status"`
	RevenueCents  int64          `json:"revenue_cents"`  // paid + shipped + delivered
	RefundedCents int64          `json:"refunded_cents"` // refunded + cancelled
	Daily         []DayTotal     `json:"daily"`
	TopProducts   []TopProduct   `json:"top_products"`
}

const soldStates = `('paid', 'approved', 'shipped', 'delivered')`

// Summary agrega ventas entre from (incluido) y to (excluido).
func (s *Store) Summary(ctx context.Context, from, to time.Time) (*Summary, error) {
	out := &Summary{From: from, To: to, OrdersByState: map[string]int{}, Daily: []DayTotal{}, TopProducts: []TopProduct{}}

	rows, err := s.pool.Query(ctx,
		`SELECT status, count(*), COALESCE(sum(total_cents), 0) FROM orders
		 WHERE created_at >= $1 AND created_at < $2 GROUP BY status`, from, to)
	if err != nil {
		return nil, err
	}
	for rows.Next() {
		var status string
		var n int
		var total int64
		if err := rows.Scan(&status, &n, &total); err != nil {
			rows.Close()
			return nil, err
		}
		out.OrdersByState[status] = n
		switch status {
		case StatusPaid, StatusApproved, StatusShipped, StatusDelivered:
			out.RevenueCents += total
		case StatusRefunded, StatusCancelled:
			out.RefundedCents += total
		}
	}
	rows.Close()
	if err := rows.Err(); err != nil {
		return nil, err
	}

	rows, err = s.pool.Query(ctx,
		`SELECT to_char(date_trunc('day', created_at), 'YYYY-MM-DD'), count(*), COALESCE(sum(total_cents), 0)
		 FROM orders WHERE status IN `+soldStates+` AND created_at >= $1 AND created_at < $2
		 GROUP BY 1 ORDER BY 1`, from, to)
	if err != nil {
		return nil, err
	}
	for rows.Next() {
		var d DayTotal
		if err := rows.Scan(&d.Day, &d.Orders, &d.RevenueCents); err != nil {
			rows.Close()
			return nil, err
		}
		out.Daily = append(out.Daily, d)
	}
	rows.Close()
	if err := rows.Err(); err != nil {
		return nil, err
	}

	rows, err = s.pool.Query(ctx,
		`SELECT oi.product_id::text, oi.name, sum(oi.quantity)::int, sum(oi.quantity * oi.unit_price_cents)
		 FROM order_items oi JOIN orders o ON o.id = oi.order_id
		 WHERE o.status IN `+soldStates+` AND o.created_at >= $1 AND o.created_at < $2
		 GROUP BY 1, 2 ORDER BY 3 DESC, 4 DESC LIMIT 10`, from, to)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	for rows.Next() {
		var t TopProduct
		if err := rows.Scan(&t.ProductID, &t.Name, &t.Units, &t.RevenueCents); err != nil {
			return nil, err
		}
		out.TopProducts = append(out.TopProducts, t)
	}
	return out, rows.Err()
}
