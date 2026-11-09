"use client";

import { useState } from "react";
import Link from "next/link";
import { keepPreviousData, useQuery } from "@tanstack/react-query";
import { salesApi } from "@/lib/api/services";
import { errorMessage } from "@/lib/api/client";
import type { OrderStatus } from "@/lib/api/types";
import { formatDate, formatMoney, STATUS_LABELS } from "@/lib/format";
import { EmptyState, ErrorState, PageLoader, Pagination } from "@/components/ui/misc";
import { PageHeader, Table, td } from "@/components/admin/admin-ui";
import { OrderStatusBadge } from "@/components/shop/order-status-badge";

const PAGE_SIZE = 15;

export default function AdminOrdersPage() {
  const [page, setPage] = useState(0);
  const [status, setStatus] = useState("");
  const query = useQuery({
    queryKey: ["orders", "admin", { status, page }],
    queryFn: () => salesApi.orders({ status, limit: PAGE_SIZE + 1, offset: page * PAGE_SIZE }),
    placeholderData: keepPreviousData,
  });
  const orders = query.data?.slice(0, PAGE_SIZE) ?? [];

  return (
    <>
      <PageHeader title="Pedidos" description="Gestiona el cumplimiento: enviar, entregar o reembolsar." />
      <select
        value={status}
        onChange={(e) => {
          setStatus(e.target.value);
          setPage(0);
        }}
        aria-label="Filtrar por estado"
        className="mb-4 h-10 rounded-lg border border-slate-300 bg-white px-3 text-sm"
      >
        <option value="">Todos los estados</option>
        {(Object.keys(STATUS_LABELS) as OrderStatus[]).map((s) => (
          <option key={s} value={s}>
            {STATUS_LABELS[s]}
          </option>
        ))}
      </select>

      {query.isLoading ? (
        <PageLoader />
      ) : query.isError ? (
        <ErrorState message={errorMessage(query.error)} onRetry={() => query.refetch()} />
      ) : orders.length === 0 ? (
        <EmptyState title="No hay pedidos" />
      ) : (
        <Table headers={[{ label: "Pedido" }, { label: "Cliente" }, { label: "Fecha" }, { label: "Estado" }, { label: "Total", className: "text-right" }]}>
          {orders.map((o) => (
            <tr key={o.id} className="hover:bg-slate-50">
              <td className={td}>
                <Link href={`/admin/orders/${o.id}`} className="font-medium text-brand-700 hover:underline">
                  #{o.id.slice(0, 8)}
                </Link>
              </td>
              <td className={td}>{o.customer_email}</td>
              <td className={`${td} text-slate-500`}>{formatDate(o.created_at)}</td>
              <td className={td}>
                <OrderStatusBadge status={o.status} />
              </td>
              <td className={`${td} text-right font-semibold`}>{formatMoney(o.total_cents, o.currency)}</td>
            </tr>
          ))}
        </Table>
      )}
      <Pagination page={page} hasNext={(query.data?.length ?? 0) > PAGE_SIZE} onChange={setPage} />
    </>
  );
}
