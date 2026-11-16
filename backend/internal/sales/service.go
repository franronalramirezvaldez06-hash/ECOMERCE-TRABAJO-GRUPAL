package sales

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"strings"

	"ecommerce/internal/apperr"
	"ecommerce/internal/catalog"
	"ecommerce/internal/payment"
	"ecommerce/internal/pricing"
)

// Catalog es lo que necesita ventas del modulo de catalogo.
type Catalog interface {
	Build(ctx context.Context, id string) (catalog.Component, error)
}

// Deps son las dependencias del servicio de ventas (inyeccion de dependencias).
type Deps struct {
	Store      *Store
	Catalog    Catalog
	Calculator pricing.Calculator
	Gateways   *payment.Registry
	Log        *slog.Logger
}

type Service struct {
	store      *Store
	catalog    Catalog
	calculator pricing.Calculator
	gateways   *payment.Registry
	log        *slog.Logger
}

func NewService(d Deps) *Service {
	return &Service{store: d.Store, catalog: d.Catalog, calculator: d.Calculator, gateways: d.Gateways, log: d.Log}
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
	if err := p.Input.validate(); err != nil {
		return nil, err
	}
	if prev, err := s.previousOrder(ctx, p); prev != nil || err != nil {
		return prev, err
	}
	gw, err := s.gateways.Get(p.Input.Gateway)
	if err != nil {
		return nil, err
	}
	order, err := s.priceOrder(ctx, p)
	if err != nil {
		return nil, err
	}
	orderID, err := s.store.createPending(ctx, order)
	if err != nil {
		return s.duplicateOrder(ctx, p, err)
	}
	// Las escrituras posteriores no deben cancelarse si el cliente se desconecta.
	bg := context.WithoutCancel(ctx)
	if err := s.charge(ctx, gw, sale{orderID: orderID, order: order}); err != nil {
		return nil, err
	}
	return s.store.Get(bg, orderID)
}

// previousOrder devuelve la orden ya creada con la misma clave de idempotencia
// (nil si no hay clave o es la primera vez).
func (s *Service) previousOrder(ctx context.Context, p CheckoutParams) (*Order, error) {
	if p.IdempotencyKey == "" {
		return nil, nil
	}
	id, err := s.store.idByIdempotencyKey(ctx, p.UserID, p.IdempotencyKey)
	if errors.Is(err, apperr.ErrNotFound) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return s.store.Get(ctx, id)
}

// duplicateOrder resuelve una peticion duplicada concurrente con la misma
// clave: devuelve la orden ganadora en vez del error de conflicto.
func (s *Service) duplicateOrder(ctx context.Context, p CheckoutParams, cause error) (*Order, error) {
	if p.IdempotencyKey == "" || !errors.Is(cause, apperr.ErrConflict) {
		return nil, cause
	}
	id, err := s.store.idByIdempotencyKey(ctx, p.UserID, p.IdempotencyKey)
	if err != nil {
		return nil, cause
	}
	return s.store.Get(ctx, id)
}

// priceOrder resuelve el carrito y calcula el total a cobrar.
func (s *Service) priceOrder(ctx context.Context, p CheckoutParams) (pendingOrder, error) {
	c, err := s.resolve(ctx, p.Input.Items)
	if err != nil {
		return pendingOrder{}, err
	}
	quote, err := s.calculator.Calculate(ctx, pricing.Request{Items: c.items, Coupon: p.Input.Coupon})
	if err != nil {
		return pendingOrder{}, err
	}
	return pendingOrder{Params: p, Items: c.items, Needs: c.needs, Quote: quote}, nil
}

// sale es una orden ya registrada como pendiente y lista para cobrarse.
type sale struct {
	orderID string
	order   pendingOrder
}

func (v sale) payment(gateway string) paymentRecord {
	return paymentRecord{OrderID: v.orderID, Gateway: gateway, AmountCents: v.order.Quote.TotalCents}
}

// charge cobra en la pasarela y registra el resultado. Un cobro rechazado se
// compensa y no devuelve error; solo fallan los problemas al registrar.
func (s *Service) charge(ctx context.Context, gw payment.Gateway, v sale) error {
	quote, p := v.order.Quote, v.order.Params
	res, chargeErr := gw.Charge(ctx, payment.ChargeRequest{
		Reference: v.orderID, AmountCents: quote.TotalCents, Currency: quote.Currency,
		Token: p.Input.PaymentToken, Email: p.Email,
	})
	bg := context.WithoutCancel(ctx)
	if chargeErr != nil {
		return s.recordFailure(bg, v.payment(gw.Name()), v.order.Needs, chargeErr)
	}
	record := v.payment(gw.Name())
	record.Status, record.ExternalRef = "approved", res.ExternalRef
	return s.recordPayment(bg, gw, record)
}

// recordFailure compensa la venta: devuelve stock y cupon y deja la orden fallida.
func (s *Service) recordFailure(ctx context.Context, p paymentRecord, needs map[string]int, cause error) error {
	p.Status, p.ErrorMessage = "error", cause.Error()
	if errors.Is(cause, payment.ErrDeclined) {
		p.Status = "declined"
	}
	if err := s.store.markFailed(ctx, p, needs); err != nil {
		err = fmt.Errorf("transaccion de venta: %w", err)
		s.log.Error("no se pudo compensar una venta fallida", "order", p.OrderID, "error", err)
		return err
	}
	return nil
}

// recordPayment marca la orden pagada. Si el cobro ya se hizo pero no se
// puede registrar, intenta devolver el dinero.
func (s *Service) recordPayment(ctx context.Context, gw payment.Gateway, p paymentRecord) error {
	err := s.store.markPaid(ctx, p)
	if err == nil {
		return nil
	}
	err = fmt.Errorf("transaccion de venta: %w", err)
	s.log.Error("cobro aprobado pero no registrado, intentando reembolso", "order", p.OrderID, "error", err)
	if refundErr := gw.Refund(ctx, p.ExternalRef, p.AmountCents); refundErr != nil {
		s.log.Error("REEMBOLSO FALLIDO, requiere revision manual", "order", p.OrderID, "external_ref", p.ExternalRef, "error", refundErr)
	}
	return err
}
