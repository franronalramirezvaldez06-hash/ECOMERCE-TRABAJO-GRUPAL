package main

import (
	"context"
	"testing"

	"ecommerce/internal/config"
	"ecommerce/internal/pricing"
)

func TestCheckoutDoesNotAddTax(t *testing.T) {
	calc := checkoutCalculator(config.Config{Currency: "PEN", ShippingCents: 500}, pricing.MapCoupons{})
	quote, err := calc.Calculate(context.Background(), pricing.Request{Items: []pricing.Item{{Name: "Producto", UnitPriceCents: 1000, Quantity: 1}}})
	if err != nil {
		t.Fatal(err)
	}
	if quote.TaxCents != 0 || quote.TotalCents != 1500 {
		t.Fatalf("cotización inesperada: %+v", quote)
	}
}
