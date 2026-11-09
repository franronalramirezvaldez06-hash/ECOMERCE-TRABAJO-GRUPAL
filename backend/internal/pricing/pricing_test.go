package pricing

import (
	"context"
	"testing"
)

func items() []Item {
	return []Item{{ProductID: "a", Name: "A", UnitPriceCents: 4000, Quantity: 2}} // 8000
}

func calc() Calculator {
	return Chain(Base{Currency: "USD"},
		WithTax(19),
		WithShipping(500, 10000),
		WithCoupons(MapCoupons{"OFF10": 10}),
	)
}

func TestFullChain(t *testing.T) {
	b, err := calc().Calculate(context.Background(), Request{Items: items(), Coupon: "off10"})
	if err != nil {
		t.Fatal(err)
	}
	// subtotal 8000, desc 800, envio 500 (7200 < 10000), impuesto 19% de 7200 = 1368
	if b.SubtotalCents != 8000 || b.DiscountCents != 800 || b.ShippingCents != 500 || b.TaxCents != 1368 {
		t.Fatalf("desglose inesperado: %+v", b)
	}
	if b.TotalCents != 8000-800+500+1368 {
		t.Fatalf("total incorrecto: %d", b.TotalCents)
	}
}

func TestFreeShippingOverThreshold(t *testing.T) {
	big := []Item{{Name: "A", UnitPriceCents: 20000, Quantity: 1}}
	b, err := calc().Calculate(context.Background(), Request{Items: big})
	if err != nil {
		t.Fatal(err)
	}
	if b.ShippingCents != 0 {
		t.Fatalf("envio debio ser gratis, fue %d", b.ShippingCents)
	}
}

func TestInvalidCoupon(t *testing.T) {
	if _, err := calc().Calculate(context.Background(), Request{Items: items(), Coupon: "NOPE"}); err == nil {
		t.Fatal("debio rechazar cupon invalido")
	}
}

func TestEmptyCart(t *testing.T) {
	if _, err := calc().Calculate(context.Background(), Request{}); err == nil {
		t.Fatal("debio rechazar carrito vacio")
	}
}

func TestDecoratorsAreOptional(t *testing.T) {
	b, err := Chain(Base{Currency: "USD"}).Calculate(context.Background(), Request{Items: items()})
	if err != nil || b.TotalCents != 8000 {
		t.Fatalf("base sola debio dar 8000: %+v %v", b, err)
	}
}
