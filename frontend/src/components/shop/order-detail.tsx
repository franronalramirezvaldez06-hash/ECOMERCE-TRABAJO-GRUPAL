import type { Order } from "@/lib/api/types";
import { formatDate, formatMoney } from "@/lib/format";
import { Card } from "@/components/ui/misc";
import { OrderStatusBadge } from "./order-status-badge";

/** Detalle de una orden (compartido entre cliente y administrador). */
export function OrderDetail({ order, actions }: { order: Order; actions?: React.ReactNode }) {
  const cur = order.currency;
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
        <Card className="divide-y divide-slate-100">
          {order.items?.map((it) => (
            <div key={`${it.product_id}-${it.name}`} className="flex justify-between gap-4 p-4 text-sm">
              <div>
                <p className="font-medium">{it.name}</p>
                <p className="text-slate-500">
                  {it.quantity} x {formatMoney(it.unit_price_cents, cur)}
                </p>
              </div>
              <p className="font-semibold">{formatMoney(it.unit_price_cents * it.quantity, cur)}</p>
            </div>
          ))}
        </Card>

        <div className="space-y-6">
          <Card className="space-y-2 p-5 text-sm">
            <h2 className="font-semibold">Totales</h2>
            <Line label="Subtotal" value={formatMoney(order.subtotal_cents, cur)} />
            {order.discount_cents > 0 && (
              <Line label={`Descuento${order.coupon ? ` (${order.coupon})` : ""}`} value={`-${formatMoney(order.discount_cents, cur)}`} />
            )}
            <Line label="Envio" value={order.shipping_cents === 0 ? "Gratis" : formatMoney(order.shipping_cents, cur)} />
            {order.tax_cents > 0 && <Line label="Impuestos" value={formatMoney(order.tax_cents, cur)} />}
            <div className="flex justify-between border-t border-slate-200 pt-2 text-base font-semibold">
              <span>Total</span>
              <span>{formatMoney(order.total_cents, cur)}</span>
            </div>
          </Card>

          <Card className="space-y-1 p-5 text-sm">
            <h2 className="mb-1 font-semibold">Envio</h2>
            <p>{order.shipping.name}</p>
            <p className="text-slate-600">{order.shipping.address}</p>
            <p className="text-slate-600">
              {order.shipping.city}, {order.shipping.country}
            </p>
            {order.shipping.phone && <p className="text-slate-600">{order.shipping.phone}</p>}
          </Card>

          {order.payment && (
            <Card className="space-y-1 p-5 text-sm">
              <h2 className="mb-1 font-semibold">Pago</h2>
              <Line label="Pasarela" value={order.payment.gateway} />
              <Line label="Estado" value={order.payment.status} />
              {order.payment.external_ref && <Line label="Referencia" value={order.payment.external_ref} />}
              {order.payment.error_message && <p className="pt-1 text-red-600">{order.payment.error_message}</p>}
            </Card>
          )}
        </div>
      </div>
    </div>
  );
}

function Line({ label, value }: { label: string; value: string }) {
  return (
    <div className="flex justify-between gap-3 text-slate-600">
      <span>{label}</span>
      <span className="break-all text-right text-slate-900">{value}</span>
    </div>
  );
}
