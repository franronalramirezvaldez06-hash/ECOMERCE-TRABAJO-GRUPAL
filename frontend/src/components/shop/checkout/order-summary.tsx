"use client";

import type { ReactNode } from "react";
import { Lock } from "lucide-react";
import { useCart } from "@/providers/cart-provider";
import { Button } from "@/components/ui/button";
import { Card } from "@/components/ui/card";
import { QuoteSummary, useQuote } from "@/components/shop/quote-summary";

function Alert({ children }: { children: ReactNode }) {
  return (
    <p role="alert" className="rounded-lg bg-red-50 p-3 text-sm text-red-700">
      {children}
    </p>
  );
}

interface OrderSummaryProps {
  failure: string | null;
  error: string | null;
  submitting: boolean;
  canPay: boolean;
}

export function OrderSummary({ failure, error, submitting, canPay }: OrderSummaryProps) {
  const { state } = useCart();
  const quote = useQuote();
  return (
    <Card className="h-fit space-y-5 p-5">
      <h2 className="font-semibold">Resumen del pedido</h2>
      <ul className="space-y-1 text-sm text-slate-600">
        {state.lines.map((l) => (
          <li key={l.productId} className="flex justify-between gap-3">
            <span className="truncate">
              {l.quantity} x {l.name}
            </span>
          </li>
        ))}
      </ul>
      <QuoteSummary quote={quote} />

      {failure && (
        <Alert>
          <strong>Pago rechazado.</strong> {failure}
        </Alert>
      )}
      {error && <Alert>{error}</Alert>}

      <Button type="submit" size="lg" className="w-full" loading={submitting} disabled={quote.isError || !canPay}>
        <Lock className="h-4 w-4" aria-hidden />
        Pagar ahora
      </Button>
      <p className="text-center text-xs text-slate-500">Si el pago falla, no se descuenta tu inventario ni tu cupon.</p>
    </Card>
  );
}
