"use client";

import { useState, type ChangeEvent, type FormEvent } from "react";
import { useMutation, useQueryClient } from "@tanstack/react-query";
import { adminApi } from "@/lib/api/services";
import { errorMessage } from "@/lib/api/client";
import type { Product } from "@/lib/api/types";
import { centsToInput, parseMoneyToCents } from "@/lib/format";
import { useToast } from "@/providers/toast-provider";

const initialForm = (product: Product | null) => ({
  sku: product?.sku ?? "",
  name: product?.name ?? "",
  description: product?.description ?? "",
  price: product ? centsToInput(product.price_cents) : "",
  stock: "0",
  discount: String(product?.bundle_discount_pct ?? 0),
  category: product?.category_id ?? "",
  image: product?.image_url ?? "",
  active: product?.active ?? true,
});

export type ProductForm = ReturnType<typeof initialForm>;

/** Valida el formulario y guarda: actualiza si hay producto, crea si no. */
function saveProduct(form: ProductForm, product: Product | null) {
  const isBundle = product?.kind === "bundle";
  const price = parseMoneyToCents(form.price);
  if (!isBundle && price === null) throw new Error("Precio invalido. Usa un formato como 12.50");
  const shared = { name: form.name, description: form.description, category_id: form.category, image_url: form.image };
  if (product) {
    const pricing = isBundle ? { bundle_discount_pct: Number(form.discount) } : { price_cents: price ?? 0 };
    return adminApi.updateProduct(product.id, { ...shared, active: form.active, ...pricing });
  }
  const stock = Number(form.stock);
  if (!Number.isInteger(stock) || stock < 0) throw new Error("El stock inicial debe ser un entero mayor o igual a 0");
  return adminApi.createProduct({ ...shared, sku: form.sku, price_cents: price ?? 0, initial_stock: stock });
}

/** Estado y guardado del formulario de producto (crear o editar). */
export function useProductForm(product: Product | null, onSaved: () => void) {
  const toast = useToast();
  const queryClient = useQueryClient();
  const [form, setForm] = useState(() => initialForm(product));
  const [error, setError] = useState<string | null>(null);

  const setField = <K extends keyof ProductForm>(key: K, value: ProductForm[K]) => setForm((f) => ({ ...f, [key]: value }));
  const bind = (key: keyof ProductForm) => (e: ChangeEvent<HTMLInputElement | HTMLTextAreaElement>) => setField(key, e.target.value);

  const save = useMutation({
    mutationFn: async () => saveProduct(form, product),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ["products"] });
      queryClient.invalidateQueries({ queryKey: ["admin"] });
      toast.success(product ? "Producto actualizado" : "Producto creado");
      onSaved();
    },
    onError: (e) => setError(errorMessage(e)),
  });

  const submit = (e: FormEvent) => {
    e.preventDefault();
    setError(null);
    save.mutate();
  };

  return { form, bind, setField, error, submit, saving: save.isPending };
}
