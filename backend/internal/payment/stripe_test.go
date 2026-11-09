package payment

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"
)

func stripeWith(t *testing.T, handler http.HandlerFunc) *Stripe {
	t.Helper()
	srv := httptest.NewServer(handler)
	t.Cleanup(srv.Close)
	s := NewStripe("sk_test_x")
	s.baseURL = srv.URL
	return s
}

func TestStripeChargeSucceeds(t *testing.T) {
	s := stripeWith(t, func(w http.ResponseWriter, r *http.Request) {
		if r.Header.Get("Authorization") != "Bearer sk_test_x" || r.Header.Get("Idempotency-Key") != "charge-o1" {
			t.Errorf("cabeceras incorrectas: %v", r.Header)
		}
		r.ParseForm()
		if r.PostForm.Get("amount") != "1500" || r.PostForm.Get("currency") != "usd" || r.PostForm.Get("payment_method") != "pm_card_visa" {
			t.Errorf("formulario incorrecto: %v", r.PostForm)
		}
		w.Write([]byte(`{"id":"pi_123","status":"succeeded"}`))
	})
	res, err := s.Charge(context.Background(), ChargeRequest{Reference: "o1", AmountCents: 1500, Currency: "USD", Token: "pm_card_visa"})
	if err != nil || res.ExternalRef != "pi_123" {
		t.Fatalf("res=%+v err=%v", res, err)
	}
}

func TestStripeCardErrorIsDeclined(t *testing.T) {
	s := stripeWith(t, func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusPaymentRequired)
		w.Write([]byte(`{"error":{"type":"card_error","code":"card_declined","message":"Tu tarjeta fue rechazada."}}`))
	})
	_, err := s.Charge(context.Background(), ChargeRequest{Reference: "o2", AmountCents: 1, Currency: "usd", Token: "pm_x"})
	if !errors.Is(err, ErrDeclined) {
		t.Fatalf("esperaba ErrDeclined, obtuvo %v", err)
	}
}

func TestStripeServerErrorIsTemporary(t *testing.T) {
	s := stripeWith(t, func(w http.ResponseWriter, r *http.Request) { w.WriteHeader(http.StatusBadGateway) })
	_, err := s.Charge(context.Background(), ChargeRequest{Reference: "o3", AmountCents: 1, Currency: "usd", Token: "pm_x"})
	if !errors.Is(err, ErrTemporary) {
		t.Fatalf("esperaba ErrTemporary, obtuvo %v", err)
	}
}

func TestStripeRequiresActionIsDeclined(t *testing.T) {
	s := stripeWith(t, func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte(`{"id":"pi_9","status":"requires_action"}`))
	})
	_, err := s.Charge(context.Background(), ChargeRequest{Reference: "o4", AmountCents: 1, Currency: "usd", Token: "pm_x"})
	if !errors.Is(err, ErrDeclined) {
		t.Fatalf("esperaba ErrDeclined, obtuvo %v", err)
	}
}

func TestStripeRefund(t *testing.T) {
	s := stripeWith(t, func(w http.ResponseWriter, r *http.Request) {
		r.ParseForm()
		if r.URL.Path != "/v1/refunds" || r.PostForm.Get("payment_intent") != "pi_123" || r.PostForm.Get("amount") != "1500" {
			t.Errorf("peticion de reembolso incorrecta: %s %v", r.URL.Path, r.PostForm)
		}
		w.Write([]byte(`{"id":"re_1"}`))
	})
	if err := s.Refund(context.Background(), "pi_123", 1500); err != nil {
		t.Fatal(err)
	}
}
