"use client";

import Link from "next/link";
import { useParams } from "next/navigation";
import { useQuery } from "@tanstack/react-query";
import { ChevronLeft } from "lucide-react";
import { salesApi } from "@/lib/api/services";
import { errorMessage } from "@/lib/api/client";
import type { Order } from "@/lib/api/types";
import { PageLoader } from "@/components/ui/spinner";
import { ErrorState } from "@/components/ui/states";
import { OrderDetail } from "@/components/shop/order-detail";
import { useOrderActionControls } from "@/components/admin/orders/order-actions";
import { useOrderActions } from "@/components/admin/orders/use-order-actions";

function OrderView({ order }: { order: Order }) {
  const controls = useOrderActionControls(order, useOrderActions(order.id));
  return (
    <div className="space-y-4">
      <Link href="/admin/orders" className="inline-flex items-center gap-1 text-sm text-slate-600 hover:text-slate-900">
        <ChevronLeft className="h-4 w-4" aria-hidden /> Pedidos
      </Link>
      <OrderDetail order={order} actions={controls.buttons} />
      {controls.dialogs}
    </div>
  );
}

export default function AdminOrderPage() {
  const { id } = useParams<{ id: string }>();
  const query = useQuery({ queryKey: ["order", id], queryFn: () => salesApi.order(id) });

  if (query.isLoading) return <PageLoader />;
  if (query.isError) return <ErrorState message={errorMessage(query.error)} onRetry={() => query.refetch()} />;
  return query.data ? <OrderView order={query.data} /> : null;
}
