"use client";

import { useQuery } from "@tanstack/react-query";
import { catalogApi } from "@/lib/api/services";
import type { Product } from "@/lib/api/types";
import { TextAreaField, TextField } from "@/components/ui/form";
import { Modal } from "@/components/ui/modal";
import { CategorySelect, FormActions, FormError } from "./form-parts";
import { useProductForm } from "./use-product-form";

/** Crear o editar un producto simple o un combo. */
export function ProductModal({ open, product, onClose }: { open: boolean; product: Product | null; onClose: () => void }) {
  const editing = product !== null;
  const isBundle = product?.kind === "bundle";
  const categories = useQuery({ queryKey: ["categories"], queryFn: catalogApi.categories, enabled: open });
  const { form, bind, setField, error, submit, saving } = useProductForm(product, onClose);

  return (
    <Modal open={open} title={editing ? "Editar producto" : "Nuevo producto"} onClose={onClose} wide>
      <form onSubmit={submit} className="space-y-4">
        <div className="grid gap-4 sm:grid-cols-2">
          <TextField label="SKU" value={form.sku} onChange={bind("sku")} required disabled={editing} />
          <TextField label="Nombre" value={form.name} onChange={bind("name")} required />
        </div>
        <TextAreaField label="Descripcion" value={form.description} onChange={bind("description")} />
        <div className="grid gap-4 sm:grid-cols-2">
          {isBundle ? (
            <TextField label="Descuento del combo (%)" type="number" min={0} max={100} value={form.discount} onChange={bind("discount")} />
          ) : (
            <TextField label="Precio (S/)" inputMode="decimal" value={form.price} onChange={bind("price")} required placeholder="12.50" />
          )}
          {!editing && <TextField label="Stock inicial" type="number" min={0} value={form.stock} onChange={bind("stock")} />}
          <CategorySelect value={form.category} onChange={(v) => setField("category", v)} categories={categories.data} />
        </div>
        <TextField label="URL de la imagen" type="url" value={form.image} onChange={bind("image")} placeholder="https://..." />
        {editing && (
          <label className="flex items-center gap-2 text-sm">
            <input type="checkbox" checked={form.active} onChange={(e) => setField("active", e.target.checked)} className="h-4 w-4 rounded border-slate-300" />
            Activo (visible y vendible en la tienda)
          </label>
        )}
        <FormError message={error} />
        <FormActions submitLabel="Guardar" loading={saving} onCancel={onClose} />
      </form>
    </Modal>
  );
}
