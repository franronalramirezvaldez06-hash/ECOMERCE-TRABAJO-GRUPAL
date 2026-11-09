"use client";

import { useState } from "react";
import Link from "next/link";
import { useParams } from "next/navigation";
import { useMutation, useQuery, useQueryClient } from "@tanstack/react-query";
import { ChevronLeft } from "lucide-react";
import { salesApi } from "@/lib/api/services";
import { ApiError, errorMessage } from "@/lib/api/client";
import { useToast } from "@/providers/toast-provider";
import { RequireAuth } from "@/components/auth/guards";
import { Button, LinkButton } from "@/components/ui/button";
import { ConfirmDialog, EmptyState, ErrorState, PageLoader } from "@/components/ui/misc";
import { OrderDetail } from "@/components/shop/order-detail";

function OrderView() {
  const { id } = useParams<{ id: string }>();
  const toast = useToast();
  const queryClient = useQueryClient();
  const [confirming, setConfirming] = useState(false);

  const query = useQuery({ queryKey: ["order", id], queryFn: () => salesApi.order(id) });
  const cancel = useMutation({
    mutationFn: () => salesApi.cancel(id),
    onSuccess: (order) => {
      queryClient.setQueryData(["order", id], order);
      queryClient.invalidateQueries({ queryKey: ["orders"] });
      queryClient.invalidateQueries({ queryKey: ["products"] });
      toast.success("Pedido cancelado. Te devolvimos el dinero.");
      setConfirming(false);
    },
    onError: (e) => {
      toast.error(errorMessage(e));
      setConfirming(false);
    },
  });

  if (query.isLoading) return <PageLoader />;
  if (query.isError) {
    if (query.error instanceof ApiError && query.error.status === 404) {
      return <EmptyState title="Pedido no encontrado" action={<LinkButton href="/orders">Ver mis pedidos</LinkButton>} />;
    }
    return <ErrorState message={errorMessage(query.error)} onRetry={() => query.refetch()} />;
  }
  const order = query.data;
  if (!order) return null;

  return (
    <div className="space-y-4">
      <Link href="/orders" className="inline-flex items-center gap-1 text-sm text-slate-600 hover:text-slate-900">
        <ChevronLeft className="h-4 w-4" aria-hidden /> Mis pedidos
      </Link>
      <OrderDetail
        order={order}
        actions={
          order.status === "paid" && (
            <Button variant="danger" size="sm" onClick={() => setConfirming(true)}>
              Cancelar pedido
            </Button>
          )
        }
      />
      <ConfirmDialog
        open={confirming}
        title="Cancelar pedido"
        message="Se reembolsara el total del pedido y los productos volveran al inventario. Esta accion no se puede deshacer."
        confirmLabel="Si, cancelar"
        danger
        loading={cancel.isPending}
        onConfirm={() => cancel.mutate()}
        onClose={() => setConfirming(false)}
      />
    </div>
  );
}

export default function OrderPage() {
  return (
    <RequireAuth>
      <OrderView />
    </RequireAuth>
  );
}
