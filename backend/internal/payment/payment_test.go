package payment

import (
	"context"
	"errors"
	"io"
	"log/slog"
	"testing"
	"time"
)

func logger() *slog.Logger { return slog.New(slog.NewTextHandler(io.Discard, nil)) }

func chain(max int64) Gateway {
	return Decorate(NewSimulated("stripe"),
		WithLogging(logger()),
		WithRetry(3, time.Millisecond),
		WithTimeout(500*time.Millisecond),
		WithAmountLimit(max),
	)
}

func TestApproved(t *testing.T) {
	res, err := chain(1000).Charge(context.Background(), ChargeRequest{Reference: "o1", AmountCents: 500, Token: "tok_ok"})
	if err != nil || res.ExternalRef == "" {
		t.Fatalf("debio aprobar: %v", err)
	}
}

func TestDeclinedIsNotRetried(t *testing.T) {
	sim := NewSimulated("paypal")
	calls := 0
	g := Decorate(countingGateway{Gateway: sim, calls: &calls}, WithRetry(3, time.Millisecond))
	_, err := g.Charge(context.Background(), ChargeRequest{Reference: "o2", Token: "tok_declined"})
	if !errors.Is(err, ErrDeclined) || calls != 1 {
		t.Fatalf("rechazo no se reintenta: err=%v calls=%d", err, calls)
	}
}

func TestRetryRecoversFromTemporaryErrors(t *testing.T) {
	_, err := chain(1000).Charge(context.Background(), ChargeRequest{Reference: "o3", AmountCents: 1, Token: "tok_flaky"})
	if err != nil {
		t.Fatalf("retry debio recuperarse: %v", err)
	}
}

func TestRetryGivesUp(t *testing.T) {
	g := Decorate(NewSimulated("mp"), WithRetry(2, time.Millisecond)) // flaky necesita 3 intentos
	_, err := g.Charge(context.Background(), ChargeRequest{Reference: "o4", Token: "tok_flaky"})
	if !errors.Is(err, ErrTemporary) {
		t.Fatalf("esperaba ErrTemporary, obtuvo %v", err)
	}
}

func TestTimeout(t *testing.T) {
	g := Decorate(NewSimulated("stripe"), WithTimeout(20*time.Millisecond))
	_, err := g.Charge(context.Background(), ChargeRequest{Reference: "o5", Token: "tok_slow"})
	if !errors.Is(err, ErrTemporary) {
		t.Fatalf("esperaba timeout como ErrTemporary, obtuvo %v", err)
	}
}

func TestAmountLimit(t *testing.T) {
	_, err := chain(1000).Charge(context.Background(), ChargeRequest{Reference: "o6", AmountCents: 5000, Token: "tok_ok"})
	if !errors.Is(err, ErrLimitExceeded) {
		t.Fatalf("esperaba ErrLimitExceeded, obtuvo %v", err)
	}
}

func TestRegistry(t *testing.T) {
	r := NewRegistry(NewSimulated("stripe"), NewSimulated("paypal"))
	if _, err := r.Get("stripe"); err != nil {
		t.Fatal(err)
	}
	if _, err := r.Get("x"); err == nil {
		t.Fatal("debio fallar con pasarela desconocida")
	}
	if got := r.Names(); len(got) != 2 || got[0] != "paypal" {
		t.Fatalf("nombres ordenados esperados, obtuvo %v", got)
	}
}

type countingGateway struct {
	Gateway
	calls *int
}

func (c countingGateway) Charge(ctx context.Context, req ChargeRequest) (ChargeResult, error) {
	*c.calls++
	return c.Gateway.Charge(ctx, req)
}
