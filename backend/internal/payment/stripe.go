package payment

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"
)

// Stripe es el adaptador REAL de Stripe (API de PaymentIntents y Refunds).
// Se activa con STRIPE_SECRET_KEY. El Token del cobro es el id de un
// PaymentMethod creado en el frontend con Stripe.js (p. ej. "pm_...").
// Usa Idempotency-Key para que un reintento nunca cobre dos veces.
type Stripe struct {
	secretKey string
	baseURL   string
	client    *http.Client
}

func NewStripe(secretKey string) *Stripe {
	return &Stripe{secretKey: secretKey, baseURL: "https://api.stripe.com", client: &http.Client{Timeout: 15 * time.Second}}
}

func (s *Stripe) Name() string { return "stripe" }

type stripeIntent struct {
	ID     string `json:"id"`
	Status string `json:"status"`
}

type stripeAPIError struct {
	Type    string `json:"type"`
	Code    string `json:"code"`
	Message string `json:"message"`
}

type stripeErrorBody struct {
	Error stripeAPIError `json:"error"`
}

func (s *Stripe) Charge(ctx context.Context, req ChargeRequest) (ChargeResult, error) {
	if req.Token == "" {
		return ChargeResult{}, fmt.Errorf("%w: token de pago requerido", ErrDeclined)
	}
	form := url.Values{}
	form.Set("amount", strconv.FormatInt(req.AmountCents, 10))
	form.Set("currency", strings.ToLower(req.Currency))
	form.Set("payment_method", req.Token)
	form.Set("confirm", "true")
	form.Set("automatic_payment_methods[enabled]", "true")
	form.Set("automatic_payment_methods[allow_redirects]", "never")
	form.Set("metadata[order_id]", req.Reference)
	if req.Email != "" {
		form.Set("receipt_email", req.Email)
	}

	var intent stripeIntent
	if err := s.post(ctx, "/v1/payment_intents", form, "charge-"+req.Reference, &intent); err != nil {
		return ChargeResult{}, err
	}
	if intent.Status != "succeeded" {
		return ChargeResult{}, fmt.Errorf("%w: el pago quedo en estado %q", ErrDeclined, intent.Status)
	}
	return ChargeResult{ExternalRef: intent.ID}, nil
}

func (s *Stripe) Refund(ctx context.Context, externalRef string, amountCents int64) error {
	form := url.Values{}
	form.Set("payment_intent", externalRef)
	form.Set("amount", strconv.FormatInt(amountCents, 10))
	var out struct {
		ID string `json:"id"`
	}
	return s.post(ctx, "/v1/refunds", form, "refund-"+externalRef, &out)
}

func (s *Stripe) post(ctx context.Context, path string, form url.Values, idempotencyKey string, out any) error {
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, s.baseURL+path, strings.NewReader(form.Encode()))
	if err != nil {
		return err
	}
	req.Header.Set("Authorization", "Bearer "+s.secretKey)
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	req.Header.Set("Idempotency-Key", idempotencyKey)

	resp, err := s.client.Do(req)
	if err != nil {
		return fmt.Errorf("%w: %v", ErrTemporary, err)
	}
	defer resp.Body.Close()
	body, _ := io.ReadAll(io.LimitReader(resp.Body, 1<<20))

	if resp.StatusCode >= 200 && resp.StatusCode < 300 {
		return json.Unmarshal(body, out)
	}
	var e stripeErrorBody
	_ = json.Unmarshal(body, &e)
	msg := e.Error.Message
	if msg == "" {
		msg = resp.Status
	}
	switch {
	case resp.StatusCode == http.StatusTooManyRequests || resp.StatusCode >= 500:
		return fmt.Errorf("%w: stripe: %s", ErrTemporary, msg)
	case resp.StatusCode == http.StatusPaymentRequired || e.Error.Type == "card_error":
		return fmt.Errorf("%w: %s", ErrDeclined, msg)
	default:
		return fmt.Errorf("stripe: %s (%s)", msg, e.Error.Code)
	}
}
