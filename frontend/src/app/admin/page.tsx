"use client";

import { useState } from "react";
import Link from "next/link";
import { useQuery } from "@tanstack/react-query";
import { adminApi } from "@/lib/api/services";
import { errorMessage } from "@/lib/api/client";
import { formatMoney, STATUS_LABELS } from "@/lib/format";
import type { OrderStatus } from "@/lib/api/types";
import { Card, ErrorState, PageLoader } from "@/components/ui/misc";
import { PageHeader, StatCard } from "@/components/admin/admin-ui";

const RANGES = [
  { days: 7, label: "7 dias" },
  { days: 30, label: "30 dias" },
  { days: 90, label: "90 dias" },
];

const isoDay = (d: Date) => d.toISOString().slice(0, 10);

export default function DashboardPage() {
  const [days, setDays] = useState(30);
  const report = useQuery({
    queryKey: ["admin", "report", days],
    queryFn: () => {
      const to = new Date();
      const from = new Date();
      from.setDate(to.getDate() - (days - 1));
      return adminApi.salesReport(isoDay(from), isoDay(to));
    },
  });
  const lowStock = useQuery({ queryKey: ["admin", "low-stock", 5], queryFn: () => adminApi.lowStock(5) });

  if (report.isLoading) return <PageLoader />;
  if (report.isError) return <ErrorState message={errorMessage(report.error)} onRetry={() => report.refetch()} />;
  const r = report.data;
  if (!r) return null;

  const paidOrders = (r.orders_by_status.paid ?? 0) + (r.orders_by_status.approved ?? 0) + (r.orders_by_status.shipped ?? 0) + (r.orders_by_status.delivered ?? 0);
  const maxDaily = Math.max(1, ...r.daily.map((d) => d.revenue_cents));

  return (
    <>
      <PageHeader
        title="Resumen de ventas"
        actions={
          <div className="inline-flex rounded-lg border border-slate-300 bg-white p-0.5" role="group" aria-label="Rango de fechas">
            {RANGES.map((range) => (
              <button
                key={range.days}
                onClick={() => setDays(range.days)}
                aria-pressed={days === range.days}
                className={`rounded-md px-3 py-1.5 text-sm font-medium ${days === range.days ? "bg-brand-600 text-white" : "text-slate-600 hover:bg-slate-100"}`}
              >
                {range.label}
              </button>
            ))}
          </div>
        }
      />

      <div className="grid gap-4 sm:grid-cols-2 xl:grid-cols-4">
        <StatCard label="Ingresos" value={formatMoney(r.revenue_cents)} tone="green" hint="Pedidos pagados, aprobados, enviados y entregados" />
        <StatCard label="Reembolsado" value={formatMoney(r.refunded_cents)} tone={r.refunded_cents > 0 ? "red" : "default"} hint="Reembolsos y cancelaciones" />
        <StatCard label="Pedidos vendidos" value={String(paidOrders)} />
        <StatCard label="Ticket promedio" value={paidOrders ? formatMoney(Math.round(r.revenue_cents / paidOrders)) : "-"} />
      </div>

      <div className="mt-6 grid gap-6 xl:grid-cols-3">
        <Card className="p-5 xl:col-span-2">
          <h2 className="font-semibold">Ingresos por dia</h2>
          {r.daily.length === 0 ? (
            <p className="py-10 text-center text-sm text-slate-500">Sin ventas en este periodo.</p>
          ) : (
            <ul className="mt-4 space-y-2" aria-label="Ingresos diarios">
              {r.daily.map((d) => (
                <li key={d.day} className="grid grid-cols-[84px_1fr_110px] items-center gap-3 text-sm">
                  <span className="text-slate-500">{d.day.slice(5)}</span>
                  <div className="h-3 overflow-hidden rounded-full bg-slate-100">
                    <div className="h-full rounded-full bg-brand-500" style={{ width: `${(d.revenue_cents / maxDaily) * 100}%` }} />
                  </div>
                  <span className="text-right font-medium">{formatMoney(d.revenue_cents)}</span>
                </li>
              ))}
            </ul>
          )}
        </Card>

        <Card className="p-5">
          <h2 className="font-semibold">Pedidos por estado</h2>
          <ul className="mt-4 space-y-2 text-sm">
            {Object.entries(r.orders_by_status).length === 0 && <li className="text-slate-500">Sin pedidos.</li>}
            {Object.entries(r.orders_by_status).map(([status, n]) => (
              <li key={status} className="flex justify-between">
                <span className="text-slate-600">{STATUS_LABELS[status as OrderStatus] ?? status}</span>
                <span className="font-medium">{n}</span>
              </li>
            ))}
          </ul>
        </Card>

        <Card className="p-5 xl:col-span-2">
          <h2 className="font-semibold">Mas vendidos</h2>
          {r.top_products.length === 0 ? (
            <p className="py-8 text-center text-sm text-slate-500">Aun no hay ventas.</p>
          ) : (
            <ol className="mt-4 divide-y divide-slate-100 text-sm">
              {r.top_products.map((p, i) => (
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

        <Card className="p-5">
          <div className="flex items-center justify-between">
            <h2 className="font-semibold">Poco stock</h2>
            <Link href="/admin/inventory" className="text-xs font-medium text-brand-700 hover:underline">
              Ver inventario
            </Link>
          </div>
          {lowStock.data && lowStock.data.length > 0 ? (
            <ul className="mt-4 space-y-2 text-sm">
              {lowStock.data.slice(0, 6).map((it) => (
                <li key={it.product_id} className="flex justify-between gap-2">
                  <span className="truncate">{it.name}</span>
                  <span className={it.quantity === 0 ? "font-semibold text-red-600" : "font-medium text-amber-600"}>{it.quantity}</span>
                </li>
              ))}
            </ul>
          ) : (
            <p className="py-6 text-center text-sm text-slate-500">Todo el inventario esta bien.</p>
          )}
        </Card>
      </div>
    </>
  );
}
