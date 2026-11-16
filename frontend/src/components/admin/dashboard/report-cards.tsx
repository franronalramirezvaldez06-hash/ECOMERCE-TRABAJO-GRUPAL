import Link from "next/link";
import type { ReactNode } from "react";
import { formatMoney, STATUS_LABELS } from "@/lib/format";
import type { LowStockItem, OrderStatus, SalesSummary } from "@/lib/api/types";
import { Card } from "@/components/ui/card";
import { StatCard } from "@/components/admin/admin-ui";

const SOLD_STATES: OrderStatus[] = ["paid", "approved", "shipped", "delivered"];

function Empty({ children }: { children: ReactNode }) {
  return <p className="py-8 text-center text-sm text-slate-500">{children}</p>;
}

export function SummaryStats({ report }: { report: SalesSummary }) {
  const sold = SOLD_STATES.reduce((n, status) => n + (report.orders_by_status[status] ?? 0), 0);
  return (
    <div className="grid gap-4 sm:grid-cols-2 xl:grid-cols-4">
      <StatCard label="Ingresos" value={formatMoney(report.revenue_cents)} tone="green" hint="Pedidos pagados, aprobados, enviados y entregados" />
      <StatCard label="Reembolsado" value={formatMoney(report.refunded_cents)} tone={report.refunded_cents > 0 ? "red" : "default"} hint="Reembolsos y cancelaciones" />
      <StatCard label="Pedidos vendidos" value={String(sold)} />
      <StatCard label="Ticket promedio" value={sold ? formatMoney(Math.round(report.revenue_cents / sold)) : "-"} />
    </div>
  );
}

export function DailyRevenueCard({ daily }: { daily: SalesSummary["daily"] }) {
  const max = Math.max(1, ...daily.map((d) => d.revenue_cents));
  return (
    <Card className="p-5 xl:col-span-2">
      <h2 className="font-semibold">Ingresos por dia</h2>
      {daily.length === 0 ? (
        <Empty>Sin ventas en este periodo.</Empty>
      ) : (
        <ul className="mt-4 space-y-2" aria-label="Ingresos diarios">
          {daily.map((d) => (
            <li key={d.day} className="grid grid-cols-[84px_1fr_110px] items-center gap-3 text-sm">
              <span className="text-slate-500">{d.day.slice(5)}</span>
              <div className="h-3 overflow-hidden rounded-full bg-slate-100">
                <div className="h-full rounded-full bg-brand-500" style={{ width: `${(d.revenue_cents / max) * 100}%` }} />
              </div>
              <span className="text-right font-medium">{formatMoney(d.revenue_cents)}</span>
            </li>
          ))}
        </ul>
      )}
    </Card>
  );
}

export function OrdersByStatusCard({ counts }: { counts: SalesSummary["orders_by_status"] }) {
  const entries = Object.entries(counts);
  return (
    <Card className="p-5">
      <h2 className="font-semibold">Pedidos por estado</h2>
      <ul className="mt-4 space-y-2 text-sm">
        {entries.length === 0 && <li className="text-slate-500">Sin pedidos.</li>}
        {entries.map(([status, n]) => (
          <li key={status} className="flex justify-between">
            <span className="text-slate-600">{STATUS_LABELS[status as OrderStatus] ?? status}</span>
            <span className="font-medium">{n}</span>
          </li>
        ))}
      </ul>
    </Card>
  );
}

export function TopProductsCard({ products }: { products: SalesSummary["top_products"] }) {
  return (
    <Card className="p-5 xl:col-span-2">
      <h2 className="font-semibold">Mas vendidos</h2>
      {products.length === 0 ? (
        <Empty>Aun no hay ventas.</Empty>
      ) : (
        <ol className="mt-4 divide-y divide-slate-100 text-sm">
          {products.map((p, i) => (
            <li key={p.product_id} className="flex items-center justify-between gap-3 py-2.5">
              <span>
                <span className="mr-2 text-slate-400">{i + 1}.</span>
                {p.name}
              </span>
              <span className="text-slate-500">
                {p.units} uds &middot; <span className="font-medium text-slate-900">{formatMoney(p.revenue_cents)}</span>
              </span>
            </li>
          ))}
        </ol>
      )}
    </Card>
  );
}

export function LowStockCard({ items }: { items: LowStockItem[] | undefined }) {
  return (
    <Card className="p-5">
      <div className="flex items-center justify-between">
        <h2 className="font-semibold">Poco stock</h2>
        <Link href="/admin/inventory" className="text-xs font-medium text-brand-700 hover:underline">
          Ver inventario
        </Link>
      </div>
      {items && items.length > 0 ? (
        <ul className="mt-4 space-y-2 text-sm">
          {items.slice(0, 6).map((it) => (
            <li key={it.product_id} className="flex justify-between gap-2">
              <span className="truncate">{it.name}</span>
              <span className={it.quantity === 0 ? "font-semibold text-red-600" : "font-medium text-amber-600"}>{it.quantity}</span>
            </li>
          ))}
        </ul>
      ) : (
        <Empty>Todo el inventario esta bien.</Empty>
      )}
    </Card>
  );
}
