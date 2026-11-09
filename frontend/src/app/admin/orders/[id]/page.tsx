"use client";

import { useState } from "react";
import Link from "next/link";
import { useParams } from "next/navigation";
import { useMutation, useQuery, useQueryClient } from "@tanstack/react-query";
import { ChevronLeft, PackageCheck, Truck, Undo2 } from "lucide-react";
import { adminApi, salesApi } from "@/lib/api/services";
import { errorMessage } from "@/lib/api/client";
import type { Order } from "@/lib/api/types";
import { useToast } from "@/providers/toast-provider";
import { Button } from "@/components/ui/button";
import { TextField } from "@/components/ui/form";
import { ErrorState, Modal, PageLoader } from "@/components/ui/misc";
import { OrderDetail } from "@/components/shop/order-detail";

export default function AdminOrderPage() {
  const { id } = useParams<{ id: string }>();
  const toast = useToast();
  const queryClient = useQueryClient();
  const [refunding, setRefunding] = useState(false);
  const [reason, setReason] = useState("");

  const query = useQuery({ queryKey: ["order", id], queryFn: () => salesApi.order(id) });

  const onDone = (message: string) => (order: Order) => {
    queryClient.setQueryData(["order", id], order);
    queryClient.invalidateQueries({ queryKey: ["orders"] });
    queryClient.invalidateQueries({ queryKey: ["products"] });
    queryClient.invalidateQueries({ queryKey: ["admin"] });
    toast.success(message);
  };
  const onError = (e: unknown) => toast.error(errorMessage(e));

  const ship = useMutation({ mutationFn: () => adminApi.ship(id), onSuccess: onDone("Pedido marcado como enviado"), onError });
  const deliver = useMutation({ mutationFn: () => adminApi.deliver(id), onSuccess: onDone("Pedido marcado como entregado"), onError });
  const refund = useMutation({
    mutationFn: () => adminApi.refund(id, reason.trim()),
    onSuccess: (o) => {
      onDone("Reembolso realizado e inventario repuesto")(o);
      setRefunding(false);
    },
    onError: (e) => {
      onError(e);
      setRefunding(false);
    },
  });

  if (query.isLoading) return <PageLoader />;
  if (query.isError) return <ErrorState message={errorMessage(query.error)} onRetry={() => query.refetch()} />;
  const order = query.data;
  if (!order) return null;

  return (
    <div className="space-y-4">
      <Link href="/admin/orders" className="inline-flex items-center gap-1 text-sm text-slate-600 hover:text-slate-900">
        <ChevronLeft className="h-4 w-4" aria-hidden /> Pedidos
      </Link>
      <OrderDetail
        order={order}
        actions={
          <div className="flex gap-2">
            {order.status === "paid" && (
              <>
                <Button size="sm" onClick={() => ship.mutate()} loading={ship.isPending}>
                  <Truck className="h-4 w-4" aria-hidden /> Marcar enviado
                </Button>
                <Button size="sm" variant="danger" onClick={() => setRefunding(true)}>
                  <Undo2 className="h-4 w-4" aria-hidden /> Reembolsar
                </Button>
              </>
            )}
            {order.status === "shipped" && (
              <Button size="sm" onClick={() => deliver.mutate()} loading={deliver.isPending}>
                <PackageCheck className="h-4 w-4" aria-hidden /> Marcar entregado
              </Button>
            )}
          </div>
        }
      />

      <Modal open={refunding} title="Reembolsar pedido" onClose={() => setRefunding(false)}>
        <p className="mb-4 text-sm text-slate-600">
          Se devolvera el total al cliente en la pasarela y los productos volveran al inventario. No se puede deshacer.
        </p>
        <TextField label="Motivo (opcional)" value={reason} onChange={(e) => setReason(e.target.value)} />
        <div className="mt-6 flex justify-end gap-2">
          <Button variant="secondary" onClick={() => setRefunding(false)}>
            Volver
          </Button>
          <Button variant="danger" onClick={() => refund.mutate()} loading={refund.isPending}>
            Confirmar reembolso
          </Button>
        </div>
      </Modal>
    </div>
  );
}
