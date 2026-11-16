import type { ReactNode } from "react";
import type { Order } from "@/lib/api/types";
import { formatDate, formatMoney } from "@/lib/format";
import { Card } from "@/components/ui/card";
import { OrderStatusBadge } from "./order-status-badge";

function Line({ label, value }: { label: string; value: string }) {
  return (
    <div className="flex justify-between gap-3 text-slate-600">
      <span>{label}</span>
      <span className="break-all text-right text-slate-900">{value}</span>
    </div>
  );
}

function ItemsCard({ order }: { order: Order }) {
  return (
    <Card className="divide-y divide-slate-100">
      {order.items?.map((it) => (
        <div key={`${it.product_id}-${it.name}`} className="flex justify-between gap-4 p-4 text-sm">
          <div>
            <p className="font-medium">{it.name}</p>
            <p className="text-slate-500">
              {it.quantity} x {formatMoney(it.unit_price_cents, order.currency)}
            </p>
          </div>
          <p className="font-semibold">{formatMoney(it.unit_price_cents * it.quantity, order.currency)}</p>
        </div>
      ))}
    </Card>
  );
}

function TotalsCard({ order }: { order: Order }) {
  const money = (cents: number) => formatMoney(cents, order.currency);
  return (
    <Card className="space-y-2 p-5 text-sm">
      <h2 className="font-semibold">Totales</h2>
      <Line label="Subtotal" value={money(order.subtotal_cents)} />
      {order.discount_cents > 0 && <Line label={`Descuento${order.coupon ? ` (${order.coupon})` : ""}`} value={`-${money(order.discount_cents)}`} />}
      <Line label="Envio" value={order.shipping_cents === 0 ? "Gratis" : money(order.shipping_cents)} />
      {order.tax_cents > 0 && <Line label="Impuestos" value={money(order.tax_cents)} />}
      <div className="flex justify-between border-t border-slate-200 pt-2 text-base font-semibold">
        <span>Total</span>
        <span>{money(order.total_cents)}</span>
      </div>
    </Card>
  );
}

function ShippingCard({ shipping }: { shipping: Order["shipping"] }) {
  return (
    <Card className="space-y-1 p-5 text-sm">
      <h2 className="mb-1 font-semibold">Envio</h2>
      <p>{shipping.name}</p>
      <p className="text-slate-600">{shipping.address}</p>
      <p className="text-slate-600">
        {shipping.city}, {shipping.country}
      </p>
      {shipping.phone && <p className="text-slate-600">{shipping.phone}</p>}
    </Card>
  );
}

function PaymentCard({ payment }: { payment: Order["payment"] }) {
  if (!payment) return null;
  return (
    <Card className="space-y-1 p-5 text-sm">
      <h2 className="mb-1 font-semibold">Pago</h2>
      <Line label="Pasarela" value={payment.gateway} />
      <Line label="Estado" value={payment.status} />
      {payment.external_ref && <Line label="Referencia" value={payment.external_ref} />}
      {payment.error_message && <p className="pt-1 text-red-600">{payment.error_message}</p>}
    </Card>
  );
}

/** Detalle de una orden (compartido entre cliente y administrador). */
export function OrderDetail({ order, actions }: { order: Order; actions?: ReactNode }) {
  return (
    <div className="space-y-6">
      <div className="flex flex-wrap items-center justify-between gap-3">
        <div>
          <h1 className="text-2xl font-bold">Pedido #{order.id.slice(0, 8)}</h1>
          <p className="text-sm text-slate-500">
            {formatDate(order.created_at)} &middot; {order.customer_email}
          </p>
        </div>
        <div className="flex items-center gap-3">
          <OrderStatusBadge status={order.status} />
          {actions}
        </div>
      </div>
      <div className="grid gap-6 lg:grid-cols-[1fr_340px]">
        <ItemsCard order={order} />
        <div className="space-y-6">
          <TotalsCard order={order} />
          <ShippingCard shipping={order.shipping} />
          <PaymentCard payment={order.payment} />
        </div>
      </div>
    </div>
  );
}
