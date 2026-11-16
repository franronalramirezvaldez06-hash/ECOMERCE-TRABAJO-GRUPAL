"use client";

import { useState } from "react";
import { PackageCheck, Truck, Undo2 } from "lucide-react";
import type { Order } from "@/lib/api/types";
import { Button } from "@/components/ui/button";
import { TextField } from "@/components/ui/form";
import { ConfirmDialog, Modal } from "@/components/ui/modal";
import type { OrderActions } from "./use-order-actions";

interface ButtonsProps {
  order: Order;
  actions: OrderActions;
  onCancel: () => void;
  onRefund: () => void;
}

/** Botones disponibles segun el estado del pedido. */
function ActionButtons({ order, actions, onCancel, onRefund }: ButtonsProps) {
  const { approve, ship, deliver } = actions;
  const refundable = order.status === "paid" || order.status === "approved";
  return (
    <div className="flex gap-2">
      {order.status === "paid" && (
        <Button size="sm" onClick={() => approve.mutate()} loading={approve.isPending}>
          <PackageCheck className="h-4 w-4" aria-hidden /> Aprobar pedido
        </Button>
      )}
      {order.status === "approved" && (
        <Button size="sm" onClick={() => ship.mutate()} loading={ship.isPending}>
          <Truck className="h-4 w-4" aria-hidden /> Marcar enviado
        </Button>
      )}
      {refundable && (
        <>
          <Button size="sm" variant="danger" onClick={onCancel}>
            Cancelar pedido
          </Button>
          <Button size="sm" variant="danger" onClick={onRefund}>
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
  );
}

function RefundModal({ refund, onClose }: { refund: OrderActions["refund"]; onClose: () => void }) {
  const [reason, setReason] = useState("");
  return (
    <Modal open title="Reembolsar pedido" onClose={onClose}>
      <p className="mb-4 text-sm text-slate-600">
        Se devolvera el total al cliente en la pasarela y los productos volveran al inventario. No se puede deshacer.
      </p>
      <TextField label="Motivo (opcional)" value={reason} onChange={(e) => setReason(e.target.value)} />
      <div className="mt-6 flex justify-end gap-2">
        <Button variant="secondary" onClick={onClose}>
          Volver
        </Button>
        <Button variant="danger" onClick={() => refund.mutate(reason, { onSettled: onClose })} loading={refund.isPending}>
          Confirmar reembolso
        </Button>
      </div>
    </Modal>
  );
}

/** Botones del pedido y los dialogos de confirmacion de cancelar y reembolsar. */
export function useOrderActionControls(order: Order, actions: OrderActions) {
  const [dialog, setDialog] = useState<"cancel" | "refund" | null>(null);
  const close = () => setDialog(null);

  const buttons = <ActionButtons order={order} actions={actions} onCancel={() => setDialog("cancel")} onRefund={() => setDialog("refund")} />;
  const dialogs = (
    <>
      <ConfirmDialog
        open={dialog === "cancel"}
        title="Cancelar pedido"
        message="Se reembolsará el pago, se repondrá el inventario y el pedido dejará de contar como ingreso."
        confirmLabel="Confirmar cancelación"
        danger
        loading={actions.cancel.isPending}
        onConfirm={() => actions.cancel.mutate(undefined, { onSuccess: close })}
        onClose={close}
      />
      {dialog === "refund" && <RefundModal refund={actions.refund} onClose={close} />}
    </>
  );
  return { buttons, dialogs };
}
