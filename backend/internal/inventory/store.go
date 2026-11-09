// Package inventory maneja existencias y su historial de movimientos.
package inventory

import (
	"context"
	"fmt"
	"time"

	"ecommerce/internal/apperr"
	"ecommerce/internal/platform/database"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"
)

// DBTX lo cumplen tanto *pgxpool.Pool como pgx.Tx, de modo que Reserve y
// Release pueden ejecutarse dentro de la transaccion de una venta.
type DBTX interface {
	Exec(ctx context.Context, sql string, args ...any) (pgconn.CommandTag, error)
	QueryRow(ctx context.Context, sql string, args ...any) pgx.Row
}

type Store struct{ pool *pgxpool.Pool }

func NewStore(pool *pgxpool.Pool) *Store { return &Store{pool: pool} }

// Levels devuelve las existencias de los productos pedidos (0 si no hay fila).
func (s *Store) Levels(ctx context.Context, productIDs []string) (map[string]int, error) {
	out := make(map[string]int, len(productIDs))
	if len(productIDs) == 0 {
		return out, nil
	}
	rows, err := s.pool.Query(ctx,
		`SELECT product_id::text, quantity FROM inventory WHERE product_id::text = ANY($1)`, productIDs)
	if err != nil {
		return nil, database.MapError(err)
	}
	defer rows.Close()
	for rows.Next() {
		var id string
		var q int
		if err := rows.Scan(&id, &q); err != nil {
			return nil, err
		}
		out[id] = q
	}
	return out, rows.Err()
}

func (s *Store) Get(ctx context.Context, productID string) (int, error) {
	var q int
	err := s.pool.QueryRow(ctx, `SELECT quantity FROM inventory WHERE product_id = $1::uuid`, productID).Scan(&q)
	return q, database.MapError(err)
}

// Adjust suma (o resta) unidades y registra el movimiento. Falla si el stock
// quedaria negativo.
func (s *Store) Adjust(ctx context.Context, productID string, delta int, reason string) (int, error) {
	if delta == 0 {
		return 0, apperr.Validation("delta no puede ser cero")
	}
	if reason == "" {
		reason = "ajuste_manual"
	}
	tx, err := s.pool.Begin(ctx)
	if err != nil {
		return 0, err
	}
	defer tx.Rollback(ctx) //nolint:errcheck

	var q int
	err = tx.QueryRow(ctx,
		`UPDATE inventory SET quantity = quantity + $2, updated_at = now()
		 WHERE product_id = $1::uuid AND quantity + $2 >= 0 RETURNING quantity`, productID, delta).Scan(&q)
	if err != nil {
		if database.MapError(err) == apperr.ErrNotFound {
			// Distinguir producto inexistente de stock insuficiente.
			if _, getErr := s.Get(ctx, productID); getErr != nil {
				return 0, getErr
			}
			return 0, fmt.Errorf("%w: el ajuste dejaria el stock negativo", apperr.ErrInsufficientStock)
		}
		return 0, database.MapError(err)
	}
	if _, err := tx.Exec(ctx,
		`INSERT INTO inventory_movements (product_id, delta, reason) VALUES ($1::uuid, $2, $3)`,
		productID, delta, reason); err != nil {
		return 0, database.MapError(err)
	}
	return q, database.MapError(tx.Commit(ctx))
}

// Set fija el stock a una cantidad exacta.
func (s *Store) Set(ctx context.Context, productID string, quantity int) (int, error) {
	if quantity < 0 {
		return 0, apperr.Validation("quantity no puede ser negativo")
	}
	current, err := s.Get(ctx, productID)
	if err != nil {
		return 0, err
	}
	if quantity == current {
		return current, nil
	}
	return s.Adjust(ctx, productID, quantity-current, "ajuste_conteo")
}

// Reserve descuenta stock de forma atomica (usar dentro de una transaccion).
func Reserve(ctx context.Context, db DBTX, productID string, qty int, orderRef string) error {
	tag, err := db.Exec(ctx,
		`UPDATE inventory SET quantity = quantity - $2, updated_at = now()
		 WHERE product_id = $1::uuid AND quantity >= $2`, productID, qty)
	if err != nil {
		return database.MapError(err)
	}
	if tag.RowsAffected() == 0 {
		return fmt.Errorf("%w: producto %s", apperr.ErrInsufficientStock, productID)
	}
	_, err = db.Exec(ctx,
		`INSERT INTO inventory_movements (product_id, delta, reason, reference) VALUES ($1::uuid, $2, 'venta', $3)`,
		productID, -qty, orderRef)
	return database.MapError(err)
}

// Release devuelve stock reservado (compensacion cuando falla el pago).
func Release(ctx context.Context, db DBTX, productID string, qty int, orderRef string) error {
	if _, err := db.Exec(ctx,
		`UPDATE inventory SET quantity = quantity + $2, updated_at = now() WHERE product_id = $1::uuid`,
		productID, qty); err != nil {
		return database.MapError(err)
	}
	_, err := db.Exec(ctx,
		`INSERT INTO inventory_movements (product_id, delta, reason, reference) VALUES ($1::uuid, $2, 'venta_revertida', $3)`,
		productID, qty, orderRef)
	return database.MapError(err)
}

type LowStockItem struct {
	ProductID string `json:"product_id"`
	SKU       string `json:"sku"`
	Name      string `json:"name"`
	Quantity  int    `json:"quantity"`
}

// LowStock lista productos activos con existencias iguales o menores al umbral.
func (s *Store) LowStock(ctx context.Context, threshold, limit int) ([]LowStockItem, error) {
	rows, err := s.pool.Query(ctx,
		`SELECT p.id::text, p.sku, p.name, i.quantity
		 FROM inventory i JOIN products p ON p.id = i.product_id
		 WHERE p.active AND i.quantity <= $1
		 ORDER BY i.quantity, p.name LIMIT $2`, threshold, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := []LowStockItem{}
	for rows.Next() {
		var it LowStockItem
		if err := rows.Scan(&it.ProductID, &it.SKU, &it.Name, &it.Quantity); err != nil {
			return nil, err
		}
		out = append(out, it)
	}
	return out, rows.Err()
}

type Movement struct {
	ID        int64     `json:"id"`
	Delta     int       `json:"delta"`
	Reason    string    `json:"reason"`
	Reference string    `json:"reference,omitempty"`
	CreatedAt time.Time `json:"created_at"`
}

// Movements devuelve el historial de un producto (mas reciente primero).
func (s *Store) Movements(ctx context.Context, productID string, limit int) ([]Movement, error) {
	rows, err := s.pool.Query(ctx,
		`SELECT id, delta, reason, reference, created_at FROM inventory_movements
		 WHERE product_id = $1::uuid ORDER BY created_at DESC, id DESC LIMIT $2`, productID, limit)
	if err != nil {
		return nil, database.MapError(err)
	}
	defer rows.Close()
	out := []Movement{}
	for rows.Next() {
		var m Movement
		if err := rows.Scan(&m.ID, &m.Delta, &m.Reason, &m.Reference, &m.CreatedAt); err != nil {
			return nil, err
		}
		out = append(out, m)
	}
	return out, rows.Err()
}
