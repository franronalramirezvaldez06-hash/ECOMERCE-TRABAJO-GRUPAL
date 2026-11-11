package sales

// Cumplimiento: aprobacion, envio y entrega de ordenes ya pagadas.

import "context"

// Approve requiere una orden pagada y no vuelve a cobrar al cliente.
func (s *Service) Approve(ctx context.Context, orderID string) (*Order, error) {
	if err := s.store.UpdateStatus(ctx, orderID, StatusPaid, StatusApproved); err != nil {
		return nil, err
	}
	return s.store.Get(ctx, orderID)
}

// MarkShipped y MarkDelivered avanzan el cumplimiento de una orden aprobada.
func (s *Service) MarkShipped(ctx context.Context, orderID string) (*Order, error) {
	if err := s.store.UpdateStatus(ctx, orderID, StatusApproved, StatusShipped); err != nil {
		return nil, err
	}
	return s.store.Get(ctx, orderID)
}

func (s *Service) MarkDelivered(ctx context.Context, orderID string) (*Order, error) {
	if err := s.store.UpdateStatus(ctx, orderID, StatusShipped, StatusDelivered); err != nil {
		return nil, err
	}
	return s.store.Get(ctx, orderID)
}
