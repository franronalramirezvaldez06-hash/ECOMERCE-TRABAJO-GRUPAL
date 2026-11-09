"use client";

import { useEffect, useState } from "react";
import Link from "next/link";
import { ShoppingBag, Tag, Trash2 } from "lucide-react";
import { formatMoney } from "@/lib/format";
import { useCart } from "@/providers/cart-provider";
import { Button, LinkButton } from "@/components/ui/button";
import { Card, EmptyState, PageLoader } from "@/components/ui/misc";
import { ProductImage } from "@/components/shop/product-image";
import { QuantityStepper } from "@/components/shop/quantity-stepper";
import { QuoteSummary, useQuote } from "@/components/shop/quote-summary";

export default function CartPage() {
  const { state, dispatch, ready } = useCart();
  const quote = useQuote();
  const [coupon, setCoupon] = useState("");

  // Refleja el cupon guardado (p. ej. al hidratar desde localStorage o quitarlo).
  useEffect(() => setCoupon(state.coupon), [state.coupon]);

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
            <div key={line.productId} className="flex gap-4 p-4">
              <ProductImage src={line.imageUrl} alt={line.name} className="h-20 w-20 shrink-0 rounded-lg" />
              <div className="flex flex-1 flex-col justify-between gap-2 sm:flex-row sm:items-center">
                <div>
                  <Link href={`/products/${line.productId}`} className="font-medium hover:text-brand-700">
                    {line.name}
                  </Link>
                  <p className="text-sm text-slate-500">{formatMoney(line.unitPriceCents)} c/u</p>
                </div>
                <div className="flex items-center gap-4">
                  <QuantityStepper
                    value={line.quantity}
                    max={line.maxQuantity}
                    onChange={(q) => dispatch({ type: "setQuantity", productId: line.productId, quantity: q })}
                    label={`Cantidad de ${line.name}`}
                  />
                  <span className="w-24 text-right font-semibold">{formatMoney(line.unitPriceCents * line.quantity)}</span>
                  <button
                    onClick={() => dispatch({ type: "remove", productId: line.productId })}
                    aria-label={`Quitar ${line.name}`}
                    className="rounded-md p-2 text-slate-400 hover:bg-red-50 hover:text-red-600"
                  >
                    <Trash2 className="h-4 w-4" />
                  </button>
                </div>
              </div>
            </div>
          ))}
        </Card>

        <Card className="h-fit space-y-5 p-5">
          <h2 className="font-semibold">Resumen</h2>

          <form
            className="flex gap-2"
            onSubmit={(e) => {
              e.preventDefault();
              dispatch({ type: "setCoupon", coupon });
            }}
          >
            <div className="relative flex-1">
              <Tag className="pointer-events-none absolute left-3 top-1/2 h-4 w-4 -translate-y-1/2 text-slate-400" aria-hidden />
              <input
                value={coupon}
                onChange={(e) => setCoupon(e.target.value)}
                placeholder="Cupon de descuento"
                aria-label="Cupon de descuento"
                className="h-10 w-full rounded-lg border border-slate-300 pl-9 pr-3 text-sm uppercase"
              />
            </div>
            <Button type="submit" variant="secondary">
              Aplicar
            </Button>
          </form>
          {state.coupon && (
            <p className="-mt-2 text-xs text-slate-500">
              Cupon aplicado: <strong>{state.coupon}</strong>{" "}
              <button
                className="text-brand-700 underline"
                onClick={() => {
                  setCoupon("");
                  dispatch({ type: "setCoupon", coupon: "" });
                }}
              >
                quitar
              </button>
            </p>
          )}

          <QuoteSummary quote={quote} />

          <LinkButton href="/checkout" size="lg" className="w-full" aria-disabled={quote.isError}>
            <ShoppingBag className="h-5 w-5" aria-hidden />
            Continuar con la compra
          </LinkButton>
          <button className="w-full text-center text-xs text-slate-500 hover:text-slate-800" onClick={() => dispatch({ type: "clear" })}>
            Vaciar carrito
          </button>
        </Card>
      </div>
    </div>
  );
}
