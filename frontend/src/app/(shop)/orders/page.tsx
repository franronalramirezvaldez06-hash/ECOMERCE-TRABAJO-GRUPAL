"use client";

import { useState } from "react";
import Link from "next/link";
import { keepPreviousData, useQuery } from "@tanstack/react-query";
import { salesApi } from "@/lib/api/services";
import { errorMessage } from "@/lib/api/client";
import { formatDate, formatMoney } from "@/lib/format";
import { RequireAuth } from "@/components/auth/guards";
import { LinkButton } from "@/components/ui/button";
import { Card, EmptyState, ErrorState, PageLoader, Pagination } from "@/components/ui/misc";
import { OrderStatusBadge } from "@/components/shop/order-status-badge";

const PAGE_SIZE = 10;

function OrdersList() {
  const [page, setPage] = useState(0);
  const query = useQuery({
    queryKey: ["orders", "mine", page],
    queryFn: () => salesApi.orders({ limit: PAGE_SIZE + 1, offset: page * PAGE_SIZE }),
    placeholderData: keepPreviousData,
  });

  if (query.isLoading) return <PageLoader />;
  if (query.isError) return <ErrorState message={errorMessage(query.error)} onRetry={() => query.refetch()} />;
  const orders = query.data?.slice(0, PAGE_SIZE) ?? [];
  const hasNext = (query.data?.length ?? 0) > PAGE_SIZE;

  return (
    <div className="space-y-6">
      <h1 className="text-2xl font-bold">Mis pedidos</h1>
      {orders.length === 0 ? (
        <EmptyState title="Aun no tienes pedidos" action={<LinkButton href="/products">Ir a comprar</LinkButton>} />
      ) : (
        <Card className="divide-y divide-slate-100">
          {orders.map((o) => (
            <Link key={o.id} href={`/orders/${o.id}`} className="flex flex-wrap items-center justify-between gap-3 p-4 hover:bg-slate-50">
              <div>
                <p className="font-medium">Pedido #{o.id.slice(0, 8)}</p>
                <p className="text-sm text-slate-500">{formatDate(o.created_at)}</p>
              </div>
              <div className="flex items-center gap-4">
                <OrderStatusBadge status={o.status} />
                <span className="w-28 text-right font-semibold">{formatMoney(o.total_cents, o.currency)}</span>
              </div>
            </Link>
          ))}
        </Card>
      )}
      <Pagination page={page} hasNext={hasNext} onChange={setPage} />
    </div>
  );
}

export default function OrdersPage() {
  return (
    <RequireAuth>
      <OrdersList />
    </RequireAuth>
  );
}
