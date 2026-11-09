// Package payment define las pasarelas de pago.
//
// PATRON DECORATOR: cada pasarela implementa Gateway y se envuelve con
// decoradores transversales (logging, reintentos, timeout, limite de monto).
// Agregar una pasarela real (Stripe, PayPal, MercadoPago) solo requiere otra
// implementacion de Gateway; los decoradores se reutilizan tal cual.
package payment

import (
	"context"
	"errors"
	"fmt"
	"sort"

	"ecommerce/internal/apperr"
)

var (
	ErrDeclined      = errors.New("pago rechazado")
	ErrTemporary     = errors.New("error temporal de la pasarela")
	ErrLimitExceeded = errors.New("monto superior al limite permitido")
)

type ChargeRequest struct {
	Reference   string // id de la orden
	AmountCents int64
	Currency    string
	Token       string // token de tarjeta/medio de pago entregado por la pasarela
	Email       string
}

type ChargeResult struct {
	ExternalRef string
}

type Gateway interface {
	Name() string
	Charge(ctx context.Context, req ChargeRequest) (ChargeResult, error)
	Refund(ctx context.Context, externalRef string, amountCents int64) error
}

// Decorator envuelve una pasarela y devuelve otra.
type Decorator func(Gateway) Gateway

// Decorate aplica los decoradores; el primero de la lista queda mas externo.
func Decorate(g Gateway, decorators ...Decorator) Gateway {
	for i := len(decorators) - 1; i >= 0; i-- {
		g = decorators[i](g)
	}
	return g
}

// Registry guarda las pasarelas disponibles por nombre.
type Registry struct{ gateways map[string]Gateway }

func NewRegistry(gateways ...Gateway) *Registry {
	r := &Registry{gateways: make(map[string]Gateway, len(gateways))}
	for _, g := range gateways {
		r.gateways[g.Name()] = g
	}
	return r
}

func (r *Registry) Get(name string) (Gateway, error) {
	g, ok := r.gateways[name]
	if !ok {
		return nil, apperr.Validation("pasarela desconocida %q (disponibles: %v)", name, r.Names())
	}
	return g, nil
}

func (r *Registry) Names() []string {
	names := make([]string, 0, len(r.gateways))
	for n := range r.gateways {
		names = append(names, n)
	}
	sort.Strings(names)
	return names
}

func (r ChargeRequest) String() string {
	return fmt.Sprintf("ref=%s monto=%d %s", r.Reference, r.AmountCents, r.Currency)
}
