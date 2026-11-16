"use client";

import { useMutation, useQueryClient } from "@tanstack/react-query";
import { adminApi, salesApi } from "@/lib/api/services";
import { errorMessage } from "@/lib/api/client";
import type { Order } from "@/lib/api/types";
import { useToast } from "@/providers/toast-provider";

/** Acciones del administrador sobre un pedido; cada una refresca los datos y avisa. */
export function useOrderActions(id: string) {
  const toast = useToast();
  const queryClient = useQueryClient();

  const feedback = (message: string) => ({
    onSuccess: (order: Order) => {
      queryClient.setQueryData(["order", id], order);
      for (const key of ["orders", "products", "admin"]) queryClient.invalidateQueries({ queryKey: [key] });
      toast.success(message);
    },
    onError: (e: unknown) => toast.error(errorMessage(e)),
  });

  return {
    approve: useMutation({ mutationFn: () => adminApi.approve(id), ...feedback("Pedido aprobado") }),
    ship: useMutation({ mutationFn: () => adminApi.ship(id), ...feedback("Pedido marcado como enviado") }),
    deliver: useMutation({ mutationFn: () => adminApi.deliver(id), ...feedback("Pedido marcado como entregado") }),
    cancel: useMutation({ mutationFn: () => salesApi.cancel(id), ...feedback("Pedido cancelado y reembolsado; inventario repuesto") }),
    refund: useMutation({
      mutationFn: (reason: string) => adminApi.refund(id, reason.trim()),
      ...feedback("Reembolso realizado e inventario repuesto"),
    }),
  };
}

export type OrderActions = ReturnType<typeof useOrderActions>;
