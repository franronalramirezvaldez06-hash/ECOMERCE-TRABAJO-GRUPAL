package payment

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"time"
)

// Cada decorador embebe Gateway: Name() y Refund() se delegan solos y solo se
// sobreescribe Charge().

// ---------------------------------------------------------------- Logging

type logging struct {
	Gateway
	log *slog.Logger
}

// WithLogging registra cada cobro: pasarela, monto, duracion y resultado.
func WithLogging(log *slog.Logger) Decorator {
	return func(g Gateway) Gateway { return logging{Gateway: g, log: log} }
}

func (l logging) Charge(ctx context.Context, req ChargeRequest) (ChargeResult, error) {
	start := time.Now()
	res, err := l.Gateway.Charge(ctx, req)
	attrs := []any{"gateway", l.Name(), "reference", req.Reference, "amount_cents", req.AmountCents, "duration", time.Since(start).String()}
	if err != nil {
		l.log.Warn("cobro fallido", append(attrs, "error", err)...)
	} else {
		l.log.Info("cobro aprobado", append(attrs, "external_ref", res.ExternalRef)...)
	}
	return res, err
}

// ------------------------------------------------------------------ Retry

type retry struct {
	Gateway
	attempts int
	backoff  time.Duration
}

// WithRetry reintenta solo ante ErrTemporary (nunca ante un rechazo).
func WithRetry(attempts int, backoff time.Duration) Decorator {
	return func(g Gateway) Gateway { return retry{Gateway: g, attempts: attempts, backoff: backoff} }
}

func (r retry) Charge(ctx context.Context, req ChargeRequest) (ChargeResult, error) {
	var lastErr error
	for i := 1; i <= r.attempts; i++ {
		res, err := r.Gateway.Charge(ctx, req)
		if err == nil {
			return res, nil
		}
		lastErr = err
		if !errors.Is(err, ErrTemporary) || i == r.attempts {
			break
		}
		select {
		case <-ctx.Done():
			return ChargeResult{}, ctx.Err()
		case <-time.After(r.backoff * time.Duration(i)):
		}
	}
	return ChargeResult{}, lastErr
}

// ---------------------------------------------------------------- Timeout

type timeout struct {
	Gateway
	d time.Duration
}

// WithTimeout limita la duracion de cada intento de cobro.
func WithTimeout(d time.Duration) Decorator {
	return func(g Gateway) Gateway { return timeout{Gateway: g, d: d} }
}

func (t timeout) Charge(ctx context.Context, req ChargeRequest) (ChargeResult, error) {
	ctx, cancel := context.WithTimeout(ctx, t.d)
	defer cancel()
	return t.Gateway.Charge(ctx, req)
}

// ------------------------------------------------------------ Limite monto

type limit struct {
	Gateway
	maxCents int64
}

// WithAmountLimit rechaza cobros por encima de un monto (control antifraude basico).
func WithAmountLimit(maxCents int64) Decorator {
	return func(g Gateway) Gateway { return limit{Gateway: g, maxCents: maxCents} }
}

func (l limit) Charge(ctx context.Context, req ChargeRequest) (ChargeResult, error) {
	if l.maxCents > 0 && req.AmountCents > l.maxCents {
		return ChargeResult{}, fmt.Errorf("%w (%d > %d)", ErrLimitExceeded, req.AmountCents, l.maxCents)
	}
	return l.Gateway.Charge(ctx, req)
}
