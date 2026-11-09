"use client";

import { useQuery } from "@tanstack/react-query";
import { salesApi } from "@/lib/api/services";
import { errorMessage } from "@/lib/api/client";
import { formatMoney } from "@/lib/format";
import { useCart } from "@/providers/cart-provider";
import { Spinner } from "@/components/ui/misc";

/** Cotizacion en vivo: el backend calcula descuento, envio e impuestos (cadena Decorator). */
export function useQuote() {
  const { state, ready } = useCart();
  const items = state.lines.map((l) => ({ product_id: l.productId, quantity: l.quantity }));
  return useQuery({
    queryKey: ["quote", items, state.coupon],
    queryFn: () => salesApi.quote(items, state.coupon),
    enabled: ready && items.length > 0,
    retry: false,
    staleTime: 10_000,
  });
}

export function QuoteSummary({ quote }: { quote: ReturnType<typeof useQuote> }) {
  if (quote.isLoading) {
    return (
      <div className="flex justify-center py-6">
        <Spinner />
      </div>
    );
  }
  if (quote.isError) {
    return (
      <p role="alert" className="rounded-lg bg-red-50 p-3 text-sm text-red-700">
        {errorMessage(quote.error)}
      </p>
    );
  }
  const q = quote.data;
  if (!q) return null;
  const cur = q.currency;
  return (
    <dl className="space-y-2 text-sm">
      <Row label="Subtotal" value={formatMoney(q.subtotal_cents, cur)} />
      {q.discount_cents > 0 && <Row label="Descuento" value={`-${formatMoney(q.discount_cents, cur)}`} tone="text-emerald-700" />}
      <Row label="Envio" value={q.shipping_cents === 0 ? "Gratis" : formatMoney(q.shipping_cents, cur)} />
      <Row label="Impuestos" value={formatMoney(q.tax_cents, cur)} />
      <div className="flex justify-between border-t border-slate-200 pt-3 text-base font-semibold">
        <dt>Total</dt>
        <dd>{formatMoney(q.total_cents, cur)}</dd>
      </div>
    </dl>
  );
}

function Row({ label, value, tone }: { label: string; value: string; tone?: string }) {
  return (
    <div className="flex justify-between text-slate-600">
      <dt>{label}</dt>
      <dd className={tone}>{value}</dd>
    </div>
  );
}
