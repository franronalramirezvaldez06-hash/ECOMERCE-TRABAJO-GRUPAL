"use client";

import { useRef, useState } from "react";
import { useRouter } from "next/navigation";
import { useQueryClient } from "@tanstack/react-query";
import { salesApi } from "@/lib/api/services";
import { ApiError, errorMessage } from "@/lib/api/client";
import type { Order, Shipping } from "@/lib/api/types";
import type { CartState } from "@/lib/cart/cart-store";
import { newIdempotencyKey } from "@/lib/format";
import { useCart } from "@/providers/cart-provider";
import { useToast } from "@/providers/toast-provider";

export interface PaymentChoice {
  gateway: string;
  token: string;
  shipping: Shipping;
}

const toPayload = (cart: CartState, { gateway, token, shipping }: PaymentChoice) => ({
  items: cart.lines.map((l) => ({ product_id: l.productId, quantity: l.quantity })),
  coupon: cart.coupon || undefined,
  gateway,
  payment_token: token,
  shipping,
});

/** Envia la compra y traduce el resultado a estado de pantalla (cargando, rechazo, error). */
export function useCheckoutSubmit() {
  const router = useRouter();
  const toast = useToast();
  const queryClient = useQueryClient();
  const { state, dispatch } = useCart();
  const [submitting, setSubmitting] = useState(false);
  const [failure, setFailure] = useState<string | null>(null);
  const [error, setError] = useState<string | null>(null);

  // Misma clave en reintentos por error de red; nueva tras cada respuesta definitiva.
  const idempotencyKey = useRef(newIdempotencyKey());
  const renewKey = () => {
    idempotencyKey.current = newIdempotencyKey();
  };

  const onPaid = (order: Order) => {
    renewKey();
    dispatch({ type: "clear" });
    queryClient.invalidateQueries({ queryKey: ["products"] });
    queryClient.invalidateQueries({ queryKey: ["orders"] });
    toast.success("Pago realizado. Tu pedido está pendiente de aprobación.");
    router.push(`/orders/${order.id}`);
  };

  const onError = (err: unknown) => {
    if (err instanceof ApiError && err.status === 402) {
      // Pago rechazado: la orden quedo "failed" y el stock se devolvio.
      renewKey();
      const order = err.body as Order | undefined;
      setFailure(order?.payment?.error_message ?? "El pago fue rechazado. Prueba con otro medio de pago.");
    } else if (err instanceof ApiError && err.status === 409) {
      renewKey();
      setError(`${err.message}. Ajusta las cantidades de tu carrito.`);
    } else {
      // Error de red u otro: se conserva la clave para reintentar sin duplicar la compra.
      setError(errorMessage(err));
    }
  };

  const submit = async (choice: PaymentChoice) => {
    setError(null);
    setFailure(null);
    setSubmitting(true);
    try {
      onPaid(await salesApi.checkout(toPayload(state, choice), idempotencyKey.current));
    } catch (err) {
      onError(err);
    } finally {
      setSubmitting(false);
    }
  };

  return { submit, submitting, failure, error };
}
