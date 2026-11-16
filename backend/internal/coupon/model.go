// Package coupon administra cupones de descuento guardados en la base de datos.
package coupon

import "time"

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
