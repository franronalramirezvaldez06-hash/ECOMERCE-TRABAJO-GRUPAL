"use client";

import { useState, type FormEvent } from "react";
import { useMutation, useQuery, useQueryClient } from "@tanstack/react-query";
import { adminApi } from "@/lib/api/services";
import { errorMessage } from "@/lib/api/client";
import type { ProductListItem } from "@/lib/api/types";
import { formatDate } from "@/lib/format";
import { useToast } from "@/providers/toast-provider";
import { SelectField, TextField } from "@/components/ui/form";
import { Modal } from "@/components/ui/modal";
import { FormActions, FormError } from "@/components/admin/products/form-parts";

const REASONS = [
  { value: "compra_proveedor", label: "Compra a proveedor" },
  { value: "ajuste_conteo", label: "Ajuste por conteo fisico" },
  { value: "merma", label: "Merma o dano" },
  { value: "devolucion", label: "Devolucion" },
];

function useAdjustStock(productId: string, onSaved: () => void) {
  const toast = useToast();
  const queryClient = useQueryClient();
  const [delta, setDelta] = useState("");
  const [reason, setReason] = useState(REASONS[0]?.value ?? "");
  const [error, setError] = useState<string | null>(null);

  const adjust = useMutation({
    mutationFn: async () => {
      const n = Number(delta);
      if (!Number.isInteger(n) || n === 0) throw new Error("Escribe un entero distinto de cero (negativo para restar)");
      return adminApi.adjustStock(productId, n, reason);
    },
    onSuccess: (res) => {
      queryClient.invalidateQueries({ queryKey: ["products"] });
      queryClient.invalidateQueries({ queryKey: ["admin"] });
      toast.success(`Stock actualizado: ${res.quantity} unidades`);
      onSaved();
    },
    onError: (e) => setError(errorMessage(e)),
  });

  const submit = (e: FormEvent) => {
    e.preventDefault();
    setError(null);
    adjust.mutate();
  };
  return { delta, setDelta, reason, setReason, error, submit, saving: adjust.isPending };
}

function MovementList({ productId }: { productId: string }) {
  const movements = useQuery({ queryKey: ["admin", "movements", productId], queryFn: () => adminApi.movements(productId, 10) });
  if (movements.isLoading) return <p className="text-sm text-slate-500">Cargando...</p>;
  if (!movements.data || movements.data.length === 0) return <p className="text-sm text-slate-500">Sin movimientos.</p>;
  return (
    <ul className="divide-y divide-slate-100 text-sm">
      {movements.data.map((m) => (
        <li key={m.id} className="flex items-center justify-between gap-3 py-2">
          <span>
            <span className={m.delta > 0 ? "font-semibold text-emerald-700" : "font-semibold text-red-600"}>{m.delta > 0 ? `+${m.delta}` : m.delta}</span>{" "}
            <span className="text-slate-600">{m.reason.replaceAll("_", " ")}</span>
          </span>
          <span className="text-xs text-slate-500">{formatDate(m.created_at)}</span>
        </li>
      ))}
    </ul>
  );
}

export function AdjustModal({ product, onClose }: { product: ProductListItem; onClose: () => void }) {
  const form = useAdjustStock(product.id, onClose);
  return (
    <Modal open title={`Ajustar stock: ${product.name}`} onClose={onClose}>
      <p className="mb-4 text-sm text-slate-600">
        Stock actual: <strong>{product.available}</strong> unidades
      </p>
      <form onSubmit={form.submit} className="space-y-4">
        <TextField label="Cantidad a sumar o restar" type="number" value={form.delta} onChange={(e) => form.setDelta(e.target.value)} placeholder="Ej: 20 o -3" required />
        <SelectField label="Motivo" value={form.reason} onChange={(e) => form.setReason(e.target.value)}>
          {REASONS.map((r) => (
            <option key={r.value} value={r.value}>
              {r.label}
            </option>
          ))}
        </SelectField>
        <FormError message={form.error} />
        <FormActions submitLabel="Aplicar" loading={form.saving} onCancel={onClose} />
      </form>
      <h3 className="mb-2 mt-6 text-sm font-semibold">Ultimos movimientos</h3>
      <MovementList productId={product.id} />
    </Modal>
  );
}
