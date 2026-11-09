// Package sales orquesta el checkout: composite (catalogo) + decorator
// (precios y pasarelas) + inventario, y persiste ordenes y pagos.
package sales

import (
	"strings"
	"time"

	"ecommerce/internal/apperr"
)

// Ciclo de vida de una orden:
//
//	pending -> paid -> shipped -> delivered
//	pending -> failed                 (pago rechazado, stock devuelto)
//	paid    -> refunded | cancelled   (reembolso por admin / cancelacion del cliente)
const (
	StatusPending   = "pending"
	StatusPaid      = "paid"
	StatusFailed    = "failed"
	StatusShipped   = "shipped"
	StatusDelivered = "delivered"
	StatusRefunded  = "refunded"
	StatusCancelled = "cancelled"
)

type ItemInput struct {
	ProductID string `json:"product_id"`
	Quantity  int    `json:"quantity"`
}

type Shipping struct {
	Name    string `json:"name"`
	Address string `json:"address"`
	City    string `json:"city"`
	Country string `json:"country"`
	Phone   string `json:"phone"`
}

type CheckoutInput struct {
	Items        []ItemInput `json:"items"`
	Coupon       string      `json:"coupon"`
	Gateway      string      `json:"gateway"`
	PaymentToken string      `json:"payment_token"`
	Shipping     Shipping    `json:"shipping"`
}

// CheckoutParams agrupa lo que no viene en el cuerpo: identidad e idempotencia.
type CheckoutParams struct {
	UserID         string
	Email          string
	IdempotencyKey string
	Input          CheckoutInput
}

type QuoteInput struct {
	Items  []ItemInput `json:"items"`
	Coupon string      `json:"coupon"`
}

type OrderItem struct {
	ProductID      string `json:"product_id"`
	Name           string `json:"name"`
	UnitPriceCents int64  `json:"unit_price_cents"`
	Quantity       int    `json:"quantity"`
}

type Payment struct {
	ID           string    `json:"id"`
	Gateway      string    `json:"gateway"`
	Status       string    `json:"status"`
	AmountCents  int64     `json:"amount_cents"`
	ExternalRef  string    `json:"external_ref,omitempty"`
	ErrorMessage string    `json:"error_message,omitempty"`
	CreatedAt    time.Time `json:"created_at"`
}

type Order struct {
	ID            string      `json:"id"`
	UserID        string      `json:"user_id,omitempty"`
	CustomerEmail string      `json:"customer_email"`
	Status        string      `json:"status"`
	Coupon        string      `json:"coupon,omitempty"`
	SubtotalCents int64       `json:"subtotal_cents"`
	DiscountCents int64       `json:"discount_cents"`
	ShippingCents int64       `json:"shipping_cents"`
	TaxCents      int64       `json:"tax_cents"`
	TotalCents    int64       `json:"total_cents"`
	Currency      string      `json:"currency"`
	Shipping      Shipping    `json:"shipping"`
	Items         []OrderItem `json:"items,omitempty"`
	Payment       *Payment    `json:"payment,omitempty"`
	CreatedAt     time.Time   `json:"created_at"`
}

func validateItems(items []ItemInput) error {
	if len(items) == 0 {
		return apperr.Validation("items no puede estar vacio")
	}
	for _, it := range items {
		if strings.TrimSpace(it.ProductID) == "" {
			return apperr.Validation("product_id es obligatorio")
		}
		if it.Quantity <= 0 {
			return apperr.Validation("quantity debe ser mayor a cero")
		}
	}
	return nil
}

func (in CheckoutInput) validate() error {
	if in.Gateway == "" {
		return apperr.Validation("gateway es obligatorio")
	}
	s := in.Shipping
	if strings.TrimSpace(s.Name) == "" || strings.TrimSpace(s.Address) == "" ||
		strings.TrimSpace(s.City) == "" || strings.TrimSpace(s.Country) == "" {
		return apperr.Validation("shipping requiere name, address, city y country")
	}
	return validateItems(in.Items)
}
