import clsx from "clsx";
import type { ReactNode } from "react";
import { errorMessage } from "@/lib/api/client";
import { Card } from "@/components/ui/card";
import { PageLoader } from "@/components/ui/spinner";
import { ErrorState } from "@/components/ui/states";

export function PageHeader({ title, description, actions }: { title: string; description?: string; actions?: ReactNode }) {
  return (
    <div className="mb-6 flex flex-wrap items-start justify-between gap-3">
      <div>
        <h1 className="text-2xl font-bold">{title}</h1>
        {description && <p className="mt-1 text-sm text-slate-500">{description}</p>}
      </div>
      {actions && <div className="flex flex-wrap gap-2">{actions}</div>}
    </div>
  );
}

export function StatCard({ label, value, hint, tone }: { label: string; value: string; hint?: string; tone?: "green" | "red" | "default" }) {
  return (
    <Card className="p-5">
      <p className="text-sm text-slate-500">{label}</p>
      <p className={clsx("mt-1 text-2xl font-bold", tone === "green" && "text-emerald-700", tone === "red" && "text-red-700")}>{value}</p>
      {hint && <p className="mt-1 text-xs text-slate-500">{hint}</p>}
    </Card>
  );
}

/** Tabla responsive con desplazamiento horizontal en pantallas pequenas. */
export function Table({ headers, children }: { headers: { label: string; className?: string }[]; children: ReactNode }) {
  return (
    <Card className="overflow-x-auto">
      <table className="w-full min-w-[640px] text-left text-sm">
        <thead className="border-b border-slate-200 bg-slate-50 text-xs uppercase tracking-wide text-slate-500">
          <tr>
            {headers.map((h) => (
              <th key={h.label} scope="col" className={clsx("px-4 py-3 font-medium", h.className)}>
                {h.label}
              </th>
            ))}
          </tr>
        </thead>
        <tbody className="divide-y divide-slate-100">{children}</tbody>
      </table>
    </Card>
  );
}

export const td = "px-4 py-3 align-middle";

export function SearchInput({ value, onChange }: { value: string; onChange: (value: string) => void }) {
  return (
    <input
      type="search"
      value={value}
      onChange={(e) => onChange(e.target.value)}
      placeholder="Buscar por nombre o SKU"
      aria-label="Buscar productos"
      className="mb-4 h-10 w-full max-w-sm rounded-lg border border-slate-300 bg-white px-3 text-sm"
    />
  );
}

interface QueryViewProps {
  query: { isLoading: boolean; isError: boolean; error: unknown; refetch: () => unknown };
  /** Se muestra cuando la consulta termino sin resultados. */
  empty: ReactNode | false;
  children: ReactNode;
}

/** Muestra cargando / error / vacio / contenido segun el estado de una consulta. */
export function QueryView({ query, empty, children }: QueryViewProps) {
  if (query.isLoading) return <PageLoader />;
  if (query.isError) return <ErrorState message={errorMessage(query.error)} onRetry={() => query.refetch()} />;
  return <>{empty || children}</>;
}
