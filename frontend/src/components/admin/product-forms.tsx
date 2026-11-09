"use client";

import { useState, type FormEvent } from "react";
import { useMutation, useQuery, useQueryClient } from "@tanstack/react-query";
import { Plus, Trash2 } from "lucide-react";
import { adminApi, catalogApi } from "@/lib/api/services";
import { errorMessage } from "@/lib/api/client";
import type { Category, Product } from "@/lib/api/types";
import { centsToInput, parseMoneyToCents } from "@/lib/format";
import { useToast } from "@/providers/toast-provider";
import { Button } from "@/components/ui/button";
import { SelectField, TextAreaField, TextField } from "@/components/ui/form";
import { Modal } from "@/components/ui/misc";

function CategorySelect({ value, onChange, categories }: { value: string; onChange: (v: string) => void; categories: Category[] | undefined }) {
  return (
    <SelectField label="Categoria" value={value} onChange={(e) => onChange(e.target.value)}>
      <option value="">Sin categoria</option>
      {categories?.map((c) => (
        <option key={c.id} value={c.id}>
          {c.name}
        </option>
      ))}
    </SelectField>
  );
}

function FormError({ message }: { message: string | null }) {
  if (!message) return null;
  return (
    <p role="alert" className="rounded-lg bg-red-50 p-3 text-sm text-red-700">
      {message}
    </p>
  );
}

/** Crear o editar un producto simple o un combo. */
export function ProductModal({ open, product, onClose }: { open: boolean; product: Product | null; onClose: () => void }) {
  const editing = product !== null;
  const isBundle = product?.kind === "bundle";
  const toast = useToast();
  const queryClient = useQueryClient();
  const categories = useQuery({ queryKey: ["categories"], queryFn: catalogApi.categories, enabled: open });

  const [form, setForm] = useState(() => ({
    sku: product?.sku ?? "",
    name: product?.name ?? "",
    description: product?.description ?? "",
    price: product ? centsToInput(product.price_cents) : "",
    stock: "0",
    discount: String(product?.bundle_discount_pct ?? 0),
    category: product?.category_id ?? "",
    image: product?.image_url ?? "",
    active: product?.active ?? true,
  }));
  const [error, setError] = useState<string | null>(null);
  const set = (k: keyof typeof form) => (e: React.ChangeEvent<HTMLInputElement | HTMLTextAreaElement>) =>
    setForm((f) => ({ ...f, [k]: e.target.value }));

  const save = useMutation({
    mutationFn: async () => {
      const price = parseMoneyToCents(form.price);
      if (!isBundle && price === null) throw new Error("Precio invalido. Usa un formato como 12.50");
      if (editing && product) {
        return adminApi.updateProduct(product.id, {
          name: form.name,
          description: form.description,
          category_id: form.category,
          image_url: form.image,
          active: form.active,
          ...(isBundle ? { bundle_discount_pct: Number(form.discount) } : { price_cents: price ?? 0 }),
        });
      }
      const stock = Number(form.stock);
      if (!Number.isInteger(stock) || stock < 0) throw new Error("El stock inicial debe ser un entero mayor o igual a 0");
      return adminApi.createProduct({
        sku: form.sku,
        name: form.name,
        description: form.description,
        price_cents: price ?? 0,
        initial_stock: stock,
        category_id: form.category,
        image_url: form.image,
      });
    },
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ["products"] });
      queryClient.invalidateQueries({ queryKey: ["admin"] });
      toast.success(editing ? "Producto actualizado" : "Producto creado");
      onClose();
    },
    onError: (e) => setError(errorMessage(e)),
  });

  const submit = (e: FormEvent) => {
    e.preventDefault();
    setError(null);
    save.mutate();
  };

  return (
    <Modal open={open} title={editing ? "Editar producto" : "Nuevo producto"} onClose={onClose} wide>
      <form onSubmit={submit} className="space-y-4">
        <div className="grid gap-4 sm:grid-cols-2">
          <TextField label="SKU" value={form.sku} onChange={set("sku")} required disabled={editing} />
          <TextField label="Nombre" value={form.name} onChange={set("name")} required />
        </div>
        <TextAreaField label="Descripcion" value={form.description} onChange={set("description")} />
        <div className="grid gap-4 sm:grid-cols-2">
          {isBundle ? (
            <TextField label="Descuento del combo (%)" type="number" min={0} max={100} value={form.discount} onChange={set("discount")} />
          ) : (
            <TextField label="Precio" inputMode="decimal" value={form.price} onChange={set("price")} required placeholder="12.50" />
          )}
          {!editing && <TextField label="Stock inicial" type="number" min={0} value={form.stock} onChange={set("stock")} />}
          <CategorySelect value={form.category} onChange={(v) => setForm((f) => ({ ...f, category: v }))} categories={categories.data} />
        </div>
        <TextField label="URL de la imagen" type="url" value={form.image} onChange={set("image")} placeholder="https://..." />
        {editing && (
          <label className="flex items-center gap-2 text-sm">
            <input type="checkbox" checked={form.active} onChange={(e) => setForm((f) => ({ ...f, active: e.target.checked }))} className="h-4 w-4 rounded border-slate-300" />
            Activo (visible y vendible en la tienda)
          </label>
        )}
        <FormError message={error} />
        <div className="flex justify-end gap-2 pt-2">
          <Button variant="secondary" onClick={onClose}>
            Cancelar
          </Button>
          <Button type="submit" loading={save.isPending}>
            Guardar
          </Button>
        </div>
      </form>
    </Modal>
  );
}

interface Row {
  key: number;
  productId: string;
  quantity: string;
}

/** Crear un combo (Composite) eligiendo productos y cantidades. */
export function BundleModal({ open, onClose }: { open: boolean; onClose: () => void }) {
  const toast = useToast();
  const queryClient = useQueryClient();
  const categories = useQuery({ queryKey: ["categories"], queryFn: catalogApi.categories, enabled: open });
  const products = useQuery({
    queryKey: ["products", "all-for-bundle"],
    queryFn: () => catalogApi.list({ limit: 200 }),
    enabled: open,
  });

  const [form, setForm] = useState({ sku: "", name: "", description: "", discount: "10", category: "", image: "" });
  const [rows, setRows] = useState<Row[]>([
    { key: 1, productId: "", quantity: "1" },
    { key: 2, productId: "", quantity: "1" },
  ]);
  const [nextKey, setNextKey] = useState(3);
  const [error, setError] = useState<string | null>(null);

  const set = (k: keyof typeof form) => (e: React.ChangeEvent<HTMLInputElement | HTMLTextAreaElement>) =>
    setForm((f) => ({ ...f, [k]: e.target.value }));
  const updateRow = (key: number, changes: Partial<Row>) => setRows((rs) => rs.map((r) => (r.key === key ? { ...r, ...changes } : r)));

  const total = rows.reduce((sum, r) => {
    const p = products.data?.find((x) => x.id === r.productId);
    return sum + (p ? p.price_cents_total * (Number(r.quantity) || 0) : 0);
  }, 0);
  const discounted = Math.round((total * (100 - (Number(form.discount) || 0))) / 100);

  const save = useMutation({
    mutationFn: () => {
      const items = rows.filter((r) => r.productId).map((r) => ({ product_id: r.productId, quantity: Number(r.quantity) }));
      if (items.length === 0) throw new Error("Agrega al menos un componente");
      if (items.some((i) => !Number.isInteger(i.quantity) || i.quantity < 1)) throw new Error("Las cantidades deben ser enteros mayores a 0");
      if (new Set(items.map((i) => i.product_id)).size !== items.length) throw new Error("Hay productos repetidos en el combo");
      return adminApi.createBundle({
        sku: form.sku,
        name: form.name,
        description: form.description,
        discount_pct: Number(form.discount) || 0,
        category_id: form.category,
        image_url: form.image,
        items,
      });
    },
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ["products"] });
      toast.success("Combo creado");
      onClose();
    },
    onError: (e) => setError(errorMessage(e)),
  });

  return (
    <Modal open={open} title="Nuevo combo" onClose={onClose} wide>
      <form
        onSubmit={(e) => {
          e.preventDefault();
          setError(null);
          save.mutate();
        }}
        className="space-y-4"
      >
        <div className="grid gap-4 sm:grid-cols-2">
          <TextField label="SKU" value={form.sku} onChange={set("sku")} required />
          <TextField label="Nombre" value={form.name} onChange={set("name")} required />
        </div>
        <TextAreaField label="Descripcion" value={form.description} onChange={set("description")} />
        <div className="grid gap-4 sm:grid-cols-2">
          <TextField label="Descuento del combo (%)" type="number" min={0} max={100} value={form.discount} onChange={set("discount")} />
          <CategorySelect value={form.category} onChange={(v) => setForm((f) => ({ ...f, category: v }))} categories={categories.data} />
        </div>
        <TextField label="URL de la imagen" type="url" value={form.image} onChange={set("image")} placeholder="https://..." />

        <fieldset className="space-y-2">
          <legend className="text-sm font-medium text-slate-700">Componentes (productos u otros combos)</legend>
          {rows.map((r) => (
            <div key={r.key} className="flex items-end gap-2">
              <div className="flex-1">
                <SelectField label="Producto" value={r.productId} onChange={(e) => updateRow(r.key, { productId: e.target.value })}>
                  <option value="">Selecciona...</option>
                  {products.data?.map((p) => (
                    <option key={p.id} value={p.id}>
                      {p.name} ({p.kind === "bundle" ? "combo" : p.sku})
                    </option>
                  ))}
                </SelectField>
              </div>
              <div className="w-24">
                <TextField label="Cant." type="number" min={1} value={r.quantity} onChange={(e) => updateRow(r.key, { quantity: e.target.value })} />
              </div>
              <button
                type="button"
                onClick={() => setRows((rs) => rs.filter((x) => x.key !== r.key))}
                disabled={rows.length <= 1}
                aria-label="Quitar componente"
                className="mb-1 rounded-md p-2 text-slate-400 hover:bg-red-50 hover:text-red-600 disabled:opacity-40"
              >
                <Trash2 className="h-4 w-4" />
              </button>
            </div>
          ))}
          <Button
            variant="ghost"
            size="sm"
            onClick={() => {
              setRows((rs) => [...rs, { key: nextKey, productId: "", quantity: "1" }]);
              setNextKey((k) => k + 1);
            }}
          >
            <Plus className="h-4 w-4" aria-hidden /> Agregar componente
          </Button>
        </fieldset>

        <p className="rounded-lg bg-slate-50 p-3 text-sm text-slate-600">
          Suma de componentes: <strong>${(total / 100).toFixed(2)}</strong> &rarr; precio del combo con descuento:{" "}
          <strong>${(discounted / 100).toFixed(2)}</strong>
        </p>
        <FormError message={error} />
        <div className="flex justify-end gap-2 pt-2">
          <Button variant="secondary" onClick={onClose}>
            Cancelar
          </Button>
          <Button type="submit" loading={save.isPending}>
            Crear combo
          </Button>
        </div>
      </form>
    </Modal>
  );
}
