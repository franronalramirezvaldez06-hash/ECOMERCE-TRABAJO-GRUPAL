package sales

// Reembolso y cancelacion de ordenes pagadas.

import (
	"context"
	"fmt"

	"ecommerce/internal/apperr"
)

// Refund devuelve el dinero de una orden pagada (y aun no enviada), repone el
// inventario y deja la orden en finalStatus ("refunded" o "cancelled").
// Si la pasarela falla, nada cambia y la orden sigue pagada.
func (s *Service) Refund(ctx context.Context, orderID, reason, finalStatus string) (*Order, error) {
	if finalStatus != StatusRefunded && finalStatus != StatusCancelled {
		return nil, apperr.Validation("estado final invalido")
	}
	req := refundRequest{OrderID: orderID, Reason: reason, FinalStatus: finalStatus}
	err := s.store.refund(ctx, req, func(p approvedPayment) error {
		return s.refundAtGateway(ctx, orderID, p)
	})
	if err != nil {
		return nil, err
	}
	return s.store.Get(ctx, orderID)
}

func (s *Service) refundAtGateway(ctx context.Context, orderID string, p approvedPayment) error {
	gw, err := s.gateways.Get(p.Gateway)
	if err != nil {
		return fmt.Errorf("%w: pasarela %q no disponible para reembolsar", apperr.ErrGateway, p.Gateway)
	}
	if err := gw.Refund(ctx, p.ExternalRef, p.AmountCents); err != nil {
		s.log.Error("la pasarela rechazo el reembolso", "order", orderID, "gateway", p.Gateway, "error", err)
		return fmt.Errorf("%w: %v", apperr.ErrGateway, err)
	}
	return nil
}
