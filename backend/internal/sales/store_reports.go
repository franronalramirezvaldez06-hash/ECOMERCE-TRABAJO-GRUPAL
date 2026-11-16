package sales

// Capa de datos de reportes de ventas.

import (
	"context"
	"time"
)

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
	if err := s.totalsByStatus(ctx, out); err != nil {
		return nil, err
	}
	if err := s.dailyTotals(ctx, out); err != nil {
		return nil, err
	}
	if err := s.topProducts(ctx, out); err != nil {
		return nil, err
	}
	return out, nil
}

func (s *Store) totalsByStatus(ctx context.Context, out *Summary) error {
	rows, err := s.pool.Query(ctx,
		`SELECT status, count(*), COALESCE(sum(total_cents), 0) FROM orders
		 WHERE created_at >= $1 AND created_at < $2 GROUP BY status`, out.From, out.To)
	if err != nil {
		return err
	}
	defer rows.Close()
	for rows.Next() {
		var status string
		var n int
		var total int64
		if err := rows.Scan(&status, &n, &total); err != nil {
			return err
		}
		out.OrdersByState[status] = n
		switch status {
		case StatusPaid, StatusApproved, StatusShipped, StatusDelivered:
			out.RevenueCents += total
		case StatusRefunded, StatusCancelled:
			out.RefundedCents += total
		}
	}
	return rows.Err()
}

func (s *Store) dailyTotals(ctx context.Context, out *Summary) error {
	rows, err := s.pool.Query(ctx,
		`SELECT to_char(date_trunc('day', created_at), 'YYYY-MM-DD'), count(*), COALESCE(sum(total_cents), 0)
		 FROM orders WHERE status IN `+soldStates+` AND created_at >= $1 AND created_at < $2
		 GROUP BY 1 ORDER BY 1`, out.From, out.To)
	if err != nil {
		return err
	}
	defer rows.Close()
	for rows.Next() {
		var d DayTotal
		if err := rows.Scan(&d.Day, &d.Orders, &d.RevenueCents); err != nil {
			return err
		}
		out.Daily = append(out.Daily, d)
	}
	return rows.Err()
}

func (s *Store) topProducts(ctx context.Context, out *Summary) error {
	rows, err := s.pool.Query(ctx,
		`SELECT oi.product_id::text, oi.name, sum(oi.quantity)::int, sum(oi.quantity * oi.unit_price_cents)
		 FROM order_items oi JOIN orders o ON o.id = oi.order_id
		 WHERE o.status IN `+soldStates+` AND o.created_at >= $1 AND o.created_at < $2
		 GROUP BY 1, 2 ORDER BY 3 DESC, 4 DESC LIMIT 10`, out.From, out.To)
	if err != nil {
		return err
	}
	defer rows.Close()
	for rows.Next() {
		var t TopProduct
		if err := rows.Scan(&t.ProductID, &t.Name, &t.Units, &t.RevenueCents); err != nil {
			return err
		}
		out.TopProducts = append(out.TopProducts, t)
	}
	return rows.Err()
}
