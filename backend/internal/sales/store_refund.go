package sales

// Capa de datos del reembolso / cancelacion.

import (
	"context"
	"errors"
	"fmt"

	"ecommerce/internal/apperr"

	"github.com/jackc/pgx/v5"
)

type refundRequest struct {
	OrderID     string
	Reason      string
	FinalStatus string // StatusRefunded o StatusCancelled
}

// approvedPayment es el cobro que hay que devolver en la pasarela.
type approvedPayment struct {
	Gateway     string
	ExternalRef string
	AmountCents int64
}

// refund cierra la orden, llama a pay (la pasarela) y repone stock y cupon,
// todo en una transaccion: si pay falla, nada cambia y la orden sigue pagada.
// El UPDATE condicional de closeOrder impide reembolsar dos veces.
func (s *Store) refund(ctx context.Context, r refundRequest, pay func(approvedPayment) error) error {
	return s.inTx(ctx, func(tx pgx.Tx) error {
		payment, err := s.closeOrder(ctx, tx, r)
		if err != nil {
			return err
		}
		if err := pay(payment); err != nil {
			return err
		}
		if err := insertPayment(ctx, tx, refundRecord(r, payment)); err != nil {
			return err
		}
		needs, err := reservedStock(ctx, tx, r.OrderID)
		if err != nil {
			return err
		}
		if err := releaseStock(ctx, tx, r.OrderID, needs); err != nil {
			return err
		}
		return releaseCoupon(ctx, tx, r.OrderID)
	})
}

func refundRecord(r refundRequest, p approvedPayment) paymentRecord {
	return paymentRecord{
		OrderID: r.OrderID, Gateway: p.Gateway, Status: "refunded",
		AmountCents: p.AmountCents, ExternalRef: p.ExternalRef, ErrorMessage: r.Reason,
	}
}

// closeOrder pasa la orden a su estado final solo si estaba pagada o aprobada
// y devuelve el cobro aprobado que corresponde reembolsar.
func (s *Store) closeOrder(ctx context.Context, tx pgx.Tx, r refundRequest) (approvedPayment, error) {
	var p approvedPayment
	err := tx.QueryRow(ctx,
		`UPDATE orders SET status = $2, updated_at = now()
		 WHERE id = $1::uuid AND status IN ('paid', 'approved') RETURNING total_cents`,
		r.OrderID, r.FinalStatus).Scan(&p.AmountCents)
	if errors.Is(err, pgx.ErrNoRows) {
		// Distinguir orden inexistente de orden en un estado no reembolsable.
		if _, getErr := s.Get(ctx, r.OrderID); getErr != nil {
			return p, getErr
		}
		return p, fmt.Errorf("%w: solo se pueden reembolsar o cancelar ordenes pagadas o aprobadas sin enviar", apperr.ErrConflict)
	}
	if err != nil {
		return p, err
	}
	err = tx.QueryRow(ctx,
		`SELECT gateway, external_ref FROM payments WHERE order_id = $1::uuid AND status = 'approved'
		 ORDER BY created_at DESC LIMIT 1`, r.OrderID).Scan(&p.Gateway, &p.ExternalRef)
	return p, err
}

// reservedStock devuelve las unidades que la orden desconto del inventario.
func reservedStock(ctx context.Context, tx pgx.Tx, orderID string) (map[string]int, error) {
	rows, err := tx.Query(ctx,
		`SELECT product_id::text, quantity FROM order_stock_items WHERE order_id = $1::uuid ORDER BY product_id`, orderID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	needs := map[string]int{}
	for rows.Next() {
		var id string
		var q int
		if err := rows.Scan(&id, &q); err != nil {
			return nil, err
		}
		needs[id] = q
	}
	return needs, rows.Err()
}
