// Package pricing calcula el total de una compra.
//
// PATRON DECORATOR: Base calcula el subtotal y cada decorador (descuento,
// envio, impuesto) envuelve a un Calculator y le agrega su parte, sin que
// los demas lo sepan. El orden de la cadena se define en un solo lugar
// (ver Chain) y se puede cambiar o ampliar sin tocar el resto del codigo.
package pricing

import (
	"context"
	"math"
	"strings"

	"ecommerce/internal/apperr"
)

type Item struct {
	ProductID      string `json:"product_id"`
	Name           string `json:"name"`
	UnitPriceCents int64  `json:"unit_price_cents"`
	Quantity       int    `json:"quantity"`
}

type Request struct {
	Items  []Item
	Coupon string
}

// Line es un renglon del desglose (los descuentos son negativos).
type Line struct {
	Concept     string `json:"concept"`
	AmountCents int64  `json:"amount_cents"`
}

type Breakdown struct {
	Currency      string `json:"currency"`
	SubtotalCents int64  `json:"subtotal_cents"`
	DiscountCents int64  `json:"discount_cents"`
	ShippingCents int64  `json:"shipping_cents"`
	TaxCents      int64  `json:"tax_cents"`
	TotalCents    int64  `json:"total_cents"`
	Lines         []Line `json:"lines"`
}

func (b *Breakdown) recalc() {
	b.TotalCents = b.SubtotalCents - b.DiscountCents + b.ShippingCents + b.TaxCents
}

// netOfDiscount es la base sobre la que se calculan envio gratis e impuesto.
func (b *Breakdown) netOfDiscount() int64 { return b.SubtotalCents - b.DiscountCents }

// Calculator es la interfaz comun al componente base y a los decoradores.
type Calculator interface {
	Calculate(ctx context.Context, req Request) (Breakdown, error)
}

// Decorator envuelve un Calculator y devuelve otro.
type Decorator func(Calculator) Calculator

// Chain aplica los decoradores sobre base; el primero de la lista queda mas
// externo (se ejecuta al final).
func Chain(base Calculator, decorators ...Decorator) Calculator {
	for i := len(decorators) - 1; i >= 0; i-- {
		base = decorators[i](base)
	}
	return base
}

// ---------------------------------------------------------------- Base

type Base struct{ Currency string }

func (b Base) Calculate(_ context.Context, req Request) (Breakdown, error) {
	if len(req.Items) == 0 {
		return Breakdown{}, apperr.Validation("la compra no tiene productos")
	}
	out := Breakdown{Currency: b.Currency}
	for _, it := range req.Items {
		if it.Quantity <= 0 {
			return Breakdown{}, apperr.Validation("la cantidad de %q debe ser mayor a cero", it.Name)
		}
		amount := it.UnitPriceCents * int64(it.Quantity)
		out.SubtotalCents += amount
		out.Lines = append(out.Lines, Line{Concept: it.Name, AmountCents: amount})
	}
	out.recalc()
	return out, nil
}

// ------------------------------------------------------ Decoradores

// CouponLookup resuelve el porcentaje de descuento de un cupon vigente.
// Debe devolver un error de validacion si el cupon no existe o no es valido.
type CouponLookup interface {
	Percent(ctx context.Context, code string) (int, error)
}

// MapCoupons es una implementacion en memoria (util en pruebas).
type MapCoupons map[string]int

func (m MapCoupons) Percent(_ context.Context, code string) (int, error) {
	pct, ok := m[code]
	if !ok {
		return 0, apperr.Validation("cupon invalido: %s", code)
	}
	return pct, nil
}

type discount struct {
	inner   Calculator
	coupons CouponLookup
}

// WithCoupons aplica un porcentaje de descuento si la solicitud trae cupon.
func WithCoupons(coupons CouponLookup) Decorator {
	return func(c Calculator) Calculator { return discount{inner: c, coupons: coupons} }
}

func (d discount) Calculate(ctx context.Context, req Request) (Breakdown, error) {
	b, err := d.inner.Calculate(ctx, req)
	if err != nil {
		return b, err
	}
	code := strings.ToUpper(strings.TrimSpace(req.Coupon))
	if code == "" {
		return b, nil
	}
	pct, err := d.coupons.Percent(ctx, code)
	if err != nil {
		return Breakdown{}, err
	}
	amount := b.SubtotalCents * int64(pct) / 100
	b.DiscountCents += amount
	b.Lines = append(b.Lines, Line{Concept: "Cupon " + code, AmountCents: -amount})
	b.recalc()
	return b, nil
}

type shipping struct {
	inner    Calculator
	flat     int64
	freeOver int64
}

// WithShipping suma un envio fijo, gratis si la compra supera freeOver.
func WithShipping(flat, freeOver int64) Decorator {
	return func(c Calculator) Calculator { return shipping{inner: c, flat: flat, freeOver: freeOver} }
}

func (s shipping) Calculate(ctx context.Context, req Request) (Breakdown, error) {
	b, err := s.inner.Calculate(ctx, req)
	if err != nil {
		return b, err
	}
	if s.freeOver > 0 && b.netOfDiscount() >= s.freeOver {
		b.Lines = append(b.Lines, Line{Concept: "Envio gratis", AmountCents: 0})
	} else {
		b.ShippingCents += s.flat
		b.Lines = append(b.Lines, Line{Concept: "Envio", AmountCents: s.flat})
	}
	b.recalc()
	return b, nil
}

type tax struct {
	inner Calculator
	rate  float64
}

// WithTax agrega un impuesto (IVA) sobre subtotal menos descuento.
func WithTax(ratePct float64) Decorator {
	return func(c Calculator) Calculator { return tax{inner: c, rate: ratePct} }
}

func (t tax) Calculate(ctx context.Context, req Request) (Breakdown, error) {
	b, err := t.inner.Calculate(ctx, req)
	if err != nil {
		return b, err
	}
	amount := int64(math.Round(float64(b.netOfDiscount()) * t.rate / 100))
	b.TaxCents += amount
	b.Lines = append(b.Lines, Line{Concept: "Impuesto", AmountCents: amount})
	b.recalc()
	return b, nil
}
