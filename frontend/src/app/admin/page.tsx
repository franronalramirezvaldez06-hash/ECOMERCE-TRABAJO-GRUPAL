"use client";

import { useState } from "react";
import { useQuery } from "@tanstack/react-query";
import { adminApi } from "@/lib/api/services";
import { errorMessage } from "@/lib/api/client";
import { PageLoader } from "@/components/ui/spinner";
import { ErrorState } from "@/components/ui/states";
import { PageHeader } from "@/components/admin/admin-ui";
import { DailyRevenueCard, LowStockCard, OrdersByStatusCard, SummaryStats, TopProductsCard } from "@/components/admin/dashboard/report-cards";

const RANGES = [
  { days: 7, label: "7 dias" },
  { days: 30, label: "30 dias" },
  { days: 90, label: "90 dias" },
];

const isoDay = (d: Date) => d.toISOString().slice(0, 10);

function fetchReport(days: number) {
  const to = new Date();
  const from = new Date();
  from.setDate(to.getDate() - (days - 1));
  return adminApi.salesReport(isoDay(from), isoDay(to));
}

function RangePicker({ days, onChange }: { days: number; onChange: (days: number) => void }) {
  return (
    <div className="inline-flex rounded-lg border border-slate-300 bg-white p-0.5" role="group" aria-label="Rango de fechas">
      {RANGES.map((range) => (
        <button
          key={range.days}
          onClick={() => onChange(range.days)}
          aria-pressed={days === range.days}
          className={`rounded-md px-3 py-1.5 text-sm font-medium ${days === range.days ? "bg-brand-600 text-white" : "text-slate-600 hover:bg-slate-100"}`}
        >
          {range.label}
        </button>
      ))}
    </div>
  );
}

export default function DashboardPage() {
  const [days, setDays] = useState(30);
  const report = useQuery({ queryKey: ["admin", "report", days], queryFn: () => fetchReport(days) });
  const lowStock = useQuery({ queryKey: ["admin", "low-stock", 5], queryFn: () => adminApi.lowStock(5) });

  if (report.isLoading) return <PageLoader />;
  if (report.isError) return <ErrorState message={errorMessage(report.error)} onRetry={() => report.refetch()} />;
  const r = report.data;
  if (!r) return null;

  return (
    <>
      <PageHeader title="Resumen de ventas" actions={<RangePicker days={days} onChange={setDays} />} />
      <SummaryStats report={r} />
      <div className="mt-6 grid gap-6 xl:grid-cols-3">
        <DailyRevenueCard daily={r.daily} />
        <OrdersByStatusCard counts={r.orders_by_status} />
        <TopProductsCard products={r.top_products} />
        <LowStockCard items={lowStock.data} />
      </div>
    </>
  );
}
