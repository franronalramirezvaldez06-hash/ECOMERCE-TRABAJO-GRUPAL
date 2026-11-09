package payment

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"sync"
	"time"
)

// Simulated es un adaptador de pasarela que NO llama a ningun servicio
// externo. Permite desarrollar y probar el flujo completo. Tokens de prueba:
//
//	tok_ok        -> aprobado (cualquier otro token tambien aprueba)
//	tok_declined  -> rechazado
//	tok_flaky     -> falla 2 veces con error temporal y luego aprueba (prueba Retry)
//	tok_slow      -> tarda 3 segundos (prueba Timeout)
type Simulated struct {
	name     string
	mu       sync.Mutex
	attempts map[string]int
}

func NewSimulated(name string) *Simulated {
	return &Simulated{name: name, attempts: make(map[string]int)}
}

func (s *Simulated) Name() string { return s.name }

func (s *Simulated) Charge(ctx context.Context, req ChargeRequest) (ChargeResult, error) {
	switch req.Token {
	case "":
		return ChargeResult{}, fmt.Errorf("%w: token de pago requerido", ErrDeclined)
	case "tok_declined":
		return ChargeResult{}, fmt.Errorf("%w: fondos insuficientes", ErrDeclined)
	case "tok_flaky":
		s.mu.Lock()
		s.attempts[req.Reference]++
		n := s.attempts[req.Reference]
		s.mu.Unlock()
		if n < 3 {
			return ChargeResult{}, fmt.Errorf("%w: intento %d", ErrTemporary, n)
		}
	case "tok_slow":
		select {
		case <-time.After(3 * time.Second):
		case <-ctx.Done():
			return ChargeResult{}, fmt.Errorf("%w: %v", ErrTemporary, ctx.Err())
		}
	}
	buf := make([]byte, 6)
	_, _ = rand.Read(buf)
	return ChargeResult{ExternalRef: s.name + "_" + hex.EncodeToString(buf)}, nil
}

func (s *Simulated) Refund(context.Context, string, int64) error { return nil }
