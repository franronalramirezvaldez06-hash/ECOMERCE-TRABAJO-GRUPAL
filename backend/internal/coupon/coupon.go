// Package coupon administra cupones de descuento guardados en la base de datos.
package coupon

import (
	"context"
	"strings"
	"time"

	"ecommerce/internal/apperr"
	"ecommerce/internal/platform/database"

	"github.com/jackc/pgx/v5/pgxpool"
)

type Coupon struct {
	Code        string     `json:"code"`
	DiscountPct int        `json:"discount_pct"`
	MaxUses     int        `json:"max_uses"`
	UsedCount   int        `json:"used_count"`
	ExpiresAt   *time.Time `json:"expires_at"`
	Active      bool       `json:"active"`
	CreatedAt   time.Time  `json:"created_at"`
}

type CreateInput struct {
	Code        string     `json:"code"`
	DiscountPct int        `json:"discount_pct"`
	MaxUses     int        `json:"max_uses"` // 0 = ilimitado
	ExpiresAt   *time.Time `json:"expires_at"`
}

type Store struct{ pool *pgxpool.Pool }

func NewStore(pool *pgxpool.Pool) *Store { return &Store{pool: pool} }

// Percent devuelve el descuento de un cupon vigente (implementa pricing.CouponLookup).
func (s *Store) Percent(ctx context.Context, code string) (int, error) {
	var pct int
	err := s.pool.QueryRow(ctx,
		`SELECT discount_pct FROM coupons
		 WHERE code = $1 AND active
		   AND (max_uses = 0 OR used_count < max_uses)
		   AND (expires_at IS NULL OR expires_at > now())`, code).Scan(&pct)
	if err != nil {
		if database.MapError(err) == apperr.ErrNotFound {
			return 0, apperr.Validation("cupon invalido, vencido o agotado: %s", code)
		}
		return 0, err
	}
	return pct, nil
}

func (s *Store) Create(ctx context.Context, in CreateInput) (Coupon, error) {
	in.Code = strings.ToUpper(strings.TrimSpace(in.Code))
	switch {
	case in.Code == "":
		return Coupon{}, apperr.Validation("code es obligatorio")
	case in.DiscountPct < 1 || in.DiscountPct > 100:
		return Coupon{}, apperr.Validation("discount_pct debe estar entre 1 y 100")
	case in.MaxUses < 0:
		return Coupon{}, apperr.Validation("max_uses no puede ser negativo")
	case in.ExpiresAt != nil && in.ExpiresAt.Before(time.Now()):
		return Coupon{}, apperr.Validation("expires_at ya paso")
	}
	var c Coupon
	err := s.pool.QueryRow(ctx,
		`INSERT INTO coupons (code, discount_pct, max_uses, expires_at) VALUES ($1, $2, $3, $4)
		 RETURNING code, discount_pct, max_uses, used_count, expires_at, active, created_at`,
		in.Code, in.DiscountPct, in.MaxUses, in.ExpiresAt).
		Scan(&c.Code, &c.DiscountPct, &c.MaxUses, &c.UsedCount, &c.ExpiresAt, &c.Active, &c.CreatedAt)
	return c, database.MapError(err)
}

func (s *Store) List(ctx context.Context) ([]Coupon, error) {
	rows, err := s.pool.Query(ctx,
		`SELECT code, discount_pct, max_uses, used_count, expires_at, active, created_at FROM coupons ORDER BY created_at DESC`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := []Coupon{}
	for rows.Next() {
		var c Coupon
		if err := rows.Scan(&c.Code, &c.DiscountPct, &c.MaxUses, &c.UsedCount, &c.ExpiresAt, &c.Active, &c.CreatedAt); err != nil {
			return nil, err
		}
		out = append(out, c)
	}
	return out, rows.Err()
}

func (s *Store) SetActive(ctx context.Context, code string, active bool) error {
	tag, err := s.pool.Exec(ctx, `UPDATE coupons SET active = $2 WHERE code = $1`, strings.ToUpper(strings.TrimSpace(code)), active)
	if err != nil {
		return database.MapError(err)
	}
	if tag.RowsAffected() == 0 {
		return apperr.ErrNotFound
	}
	return nil
}
