package sales

// Reembolso y cancelacion de ordenes pagadas.

import (
	"context"
	"errors"
	"fmt"

	"ecommerce/internal/apperr"
	"ecommerce/internal/inventory"
	"ecommerce/internal/platform/database"

	"github.com/jackc/pgx/v5"
)

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
		`UPDATE orders SET status = $2, updated_at = now() WHERE id = $1::uuid AND status IN ('paid', 'approved') RETURNING total_cents`,
		orderID, finalStatus).Scan(&total)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			if _, getErr := s.store.Get(ctx, orderID); getErr != nil {
				return nil, getErr
			}
			return nil, fmt.Errorf("%w: solo se pueden reembolsar o cancelar ordenes pagadas o aprobadas sin enviar", apperr.ErrConflict)
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
	if _, err := tx.Exec(ctx, `UPDATE coupons SET used_count = GREATEST(used_count - 1, 0)
		WHERE code = (SELECT NULLIF(coupon, '') FROM orders WHERE id = $1::uuid)`, orderID); err != nil {
		return nil, database.MapError(err)
	}
	if err := tx.Commit(ctx); err != nil {
		return nil, database.MapError(err)
	}
	return s.store.Get(ctx, orderID)
}
