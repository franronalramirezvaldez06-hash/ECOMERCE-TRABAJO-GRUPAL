"use client";

import { useQuery } from "@tanstack/react-query";
import { Plus, Trash2 } from "lucide-react";
import { catalogApi } from "@/lib/api/services";
import type { ProductListItem } from "@/lib/api/types";
import { formatMoney } from "@/lib/format";
import { Button } from "@/components/ui/button";
import { SelectField, TextAreaField, TextField } from "@/components/ui/form";
import { Modal } from "@/components/ui/modal";
import { CategorySelect, FormActions, FormError } from "./form-parts";
import { useBundleForm, type BundleRow } from "./use-bundle-form";

type Products = ProductListItem[] | undefined;

interface ComponentRowProps {
  row: BundleRow;
  products: Products;
  onChange: (changes: Partial<BundleRow>) => void;
  onRemove?: () => void;
}

function ComponentRow({ row, products, onChange, onRemove }: ComponentRowProps) {
  return (
    <div className="flex items-end gap-2">
      <div className="flex-1">
        <SelectField label="Producto" value={row.productId} onChange={(e) => onChange({ productId: e.target.value })}>
          <option value="">Selecciona...</option>
          {products?.map((p) => (
            <option key={p.id} value={p.id}>
              {p.name} ({p.kind === "bundle" ? "combo" : p.sku})
            </option>
          ))}
        </SelectField>
      </div>
      <div className="w-24">
        <TextField label="Cant." type="number" min={1} value={row.quantity} onChange={(e) => onChange({ quantity: e.target.value })} />
      </div>
      <button
        type="button"
        onClick={onRemove}
        disabled={!onRemove}
        aria-label="Quitar componente"
        className="mb-1 rounded-md p-2 text-slate-400 hover:bg-red-50 hover:text-red-600 disabled:opacity-40"
      >
        <Trash2 className="h-4 w-4" />
      </button>
    </div>
  );
}

/** Suma de los componentes y precio final con el descuento del combo. */
function PricePreview({ rows, products, discount }: { rows: BundleRow[]; products: Products; discount: string }) {
  const total = rows.reduce((sum, r) => {
    const p = products?.find((x) => x.id === r.productId);
    return sum + (p ? p.price_cents_total * (Number(r.quantity) || 0) : 0);
  }, 0);
  const discounted = Math.round((total * (100 - (Number(discount) || 0))) / 100);
  return (
    <p className="rounded-lg bg-slate-50 p-3 text-sm text-slate-600">
      Suma de componentes: <strong>{formatMoney(total)}</strong> &rarr; precio del combo con descuento: <strong>{formatMoney(discounted)}</strong>
    </p>
  );
}

/** Crear un combo (Composite) eligiendo productos y cantidades. */
export function BundleModal({ open, onClose }: { open: boolean; onClose: () => void }) {
  const categories = useQuery({ queryKey: ["categories"], queryFn: catalogApi.categories, enabled: open });
  const products = useQuery({ queryKey: ["products", "all-for-bundle"], queryFn: () => catalogApi.list({ limit: 200 }), enabled: open });
  const { form, bind, setField, components, error, submit, saving } = useBundleForm(onClose);
  const { rows } = components;

  return (
    <Modal open={open} title="Nuevo combo" onClose={onClose} wide>
      <form onSubmit={submit} className="space-y-4">
        <div className="grid gap-4 sm:grid-cols-2">
          <TextField label="SKU" value={form.sku} onChange={bind("sku")} required />
          <TextField label="Nombre" value={form.name} onChange={bind("name")} required />
        </div>
        <TextAreaField label="Descripcion" value={form.description} onChange={bind("description")} />
        <div className="grid gap-4 sm:grid-cols-2">
          <TextField label="Descuento del combo (%)" type="number" min={0} max={100} value={form.discount} onChange={bind("discount")} />
          <CategorySelect value={form.category} onChange={(v) => setField("category", v)} categories={categories.data} />
        </div>
        <TextField label="URL de la imagen" type="url" value={form.image} onChange={bind("image")} placeholder="https://..." />

        <fieldset className="space-y-2">
          <legend className="text-sm font-medium text-slate-700">Componentes (productos u otros combos)</legend>
          {rows.map((r) => (
            <ComponentRow
              key={r.key}
              row={r}
              products={products.data}
              onChange={(changes) => components.update(r.key, changes)}
              onRemove={rows.length > 1 ? () => components.remove(r.key) : undefined}
            />
          ))}
          <Button variant="ghost" size="sm" onClick={components.add}>
            <Plus className="h-4 w-4" aria-hidden /> Agregar componente
          </Button>
        </fieldset>

        <PricePreview rows={rows} products={products.data} discount={form.discount} />
        <FormError message={error} />
        <FormActions submitLabel="Crear combo" loading={saving} onCancel={onClose} />
      </form>
    </Modal>
  );
}
