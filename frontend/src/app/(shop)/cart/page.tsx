"use client";

import { ShoppingBag } from "lucide-react";
import { useCart } from "@/providers/cart-provider";
import { LinkButton } from "@/components/ui/button";
import { Card } from "@/components/ui/card";
import { PageLoader } from "@/components/ui/spinner";
import { EmptyState } from "@/components/ui/states";
import { CartLineRow } from "@/components/shop/cart/cart-line-row";
import { CouponForm } from "@/components/shop/cart/coupon-form";
import { QuoteSummary, useQuote } from "@/components/shop/quote-summary";

function CartSummary() {
  const { dispatch } = useCart();
  const quote = useQuote();
  return (
    <Card className="h-fit space-y-5 p-5">
      <h2 className="font-semibold">Resumen</h2>
      <CouponForm />
      <QuoteSummary quote={quote} />
      <LinkButton href="/checkout" size="lg" className="w-full" aria-disabled={quote.isError}>
        <ShoppingBag className="h-5 w-5" aria-hidden />
        Continuar con la compra
      </LinkButton>
      <button className="w-full text-center text-xs text-slate-500 hover:text-slate-800" onClick={() => dispatch({ type: "clear" })}>
        Vaciar carrito
      </button>
    </Card>
  );
}

export default function CartPage() {
  const { state, ready } = useCart();

  if (!ready) return <PageLoader />;
  if (state.lines.length === 0) {
    return (
      <EmptyState
        title="Tu carrito esta vacio"
        description="Agrega productos para continuar con la compra."
        action={<LinkButton href="/products">Explorar productos</LinkButton>}
      />
    );
  }

  return (
    <div className="space-y-6">
      <h1 className="text-2xl font-bold">Carrito</h1>
      <div className="grid gap-6 lg:grid-cols-[1fr_380px]">
        <Card className="divide-y divide-slate-100">
          {state.lines.map((line) => (
            <CartLineRow key={line.productId} line={line} />
          ))}
        </Card>
        <CartSummary />
      </div>
    </div>
  );
}
