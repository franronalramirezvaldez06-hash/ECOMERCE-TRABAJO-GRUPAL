"use client";

import { useState, type ChangeEvent, type FormEvent } from "react";
import { useMutation, useQueryClient } from "@tanstack/react-query";
import { adminApi } from "@/lib/api/services";
import { errorMessage } from "@/lib/api/client";
import { useToast } from "@/providers/toast-provider";

export interface BundleRow {
  key: number;
  productId: string;
  quantity: string;
}

const EMPTY_FORM = { sku: "", name: "", description: "", discount: "10", category: "", image: "" };
export type BundleForm = typeof EMPTY_FORM;

/** Convierte las filas en items validos o lanza el error que vera el usuario. */
function toItems(rows: BundleRow[]) {
  const items = rows.filter((r) => r.productId).map((r) => ({ product_id: r.productId, quantity: Number(r.quantity) }));
  if (items.length === 0) throw new Error("Agrega al menos un componente");
  if (items.some((i) => !Number.isInteger(i.quantity) || i.quantity < 1)) throw new Error("Las cantidades deben ser enteros mayores a 0");
  if (new Set(items.map((i) => i.product_id)).size !== items.length) throw new Error("Hay productos repetidos en el combo");
  return items;
}

function useBundleRows() {
  const [rows, setRows] = useState<BundleRow[]>([
    { key: 1, productId: "", quantity: "1" },
    { key: 2, productId: "", quantity: "1" },
  ]);
  const [nextKey, setNextKey] = useState(3);

  const update = (key: number, changes: Partial<BundleRow>) => setRows((rs) => rs.map((r) => (r.key === key ? { ...r, ...changes } : r)));
  const remove = (key: number) => setRows((rs) => rs.filter((r) => r.key !== key));
  const add = () => {
    setRows((rs) => [...rs, { key: nextKey, productId: "", quantity: "1" }]);
    setNextKey((k) => k + 1);
  };
  return { rows, update, remove, add };
}

/** Estado y guardado del formulario de combo. */
export function useBundleForm(onSaved: () => void) {
  const toast = useToast();
  const queryClient = useQueryClient();
  const [form, setForm] = useState(EMPTY_FORM);
  const [error, setError] = useState<string | null>(null);
  const components = useBundleRows();

  const setField = (key: keyof BundleForm, value: string) => setForm((f) => ({ ...f, [key]: value }));
  const bind = (key: keyof BundleForm) => (e: ChangeEvent<HTMLInputElement | HTMLTextAreaElement>) => setField(key, e.target.value);

  const save = useMutation({
    mutationFn: async () =>
      adminApi.createBundle({
        sku: form.sku,
        name: form.name,
        description: form.description,
        discount_pct: Number(form.discount) || 0,
        category_id: form.category,
        image_url: form.image,
        items: toItems(components.rows),
      }),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ["products"] });
      toast.success("Combo creado");
      onSaved();
    },
    onError: (e) => setError(errorMessage(e)),
  });

  const submit = (e: FormEvent) => {
    e.preventDefault();
    setError(null);
    save.mutate();
  };

  return { form, bind, setField, components, error, submit, saving: save.isPending };
}
