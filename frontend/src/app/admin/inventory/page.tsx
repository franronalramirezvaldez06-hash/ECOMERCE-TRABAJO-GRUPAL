"use client";

import { useEffect, useState, type FormEvent } from "react";
import { keepPreviousData, useMutation, useQuery, useQueryClient } from "@tanstack/react-query";
import { AlertTriangle } from "lucide-react";
import { adminApi, catalogApi } from "@/lib/api/services";
import { errorMessage } from "@/lib/api/client";
import type { ProductListItem } from "@/lib/api/types";
import { formatDate } from "@/lib/format";
import { useToast } from "@/providers/toast-provider";
import { Button } from "@/components/ui/button";
import { SelectField, TextField } from "@/components/ui/form";
import { Badge, Card, EmptyState, ErrorState, Modal, PageLoader, Pagination } from "@/components/ui/misc";
import { PageHeader, Table, td } from "@/components/admin/admin-ui";

const PAGE_SIZE = 15;
const REASONS = [
  { value: "compra_proveedor", label: "Compra a proveedor" },
  { value: "ajuste_conteo", label: "Ajuste por conteo fisico" },
  { value: "merma", label: "Merma o dano" },
  { value: "devolucion", label: "Devolucion" },
];

function AdjustModal({ product, onClose }: { product: ProductListItem; onClose: () => void }) {
  const toast = useToast();
  const queryClient = useQueryClient();
  const [delta, setDelta] = useState("");
  const [reason, setReason] = useState(REASONS[0]?.value ?? "");
  const [error, setError] = useState<string | null>(null);

  const movements = useQuery({ queryKey: ["admin", "movements", product.id], queryFn: () => adminApi.movements(product.id, 10) });

  const adjust = useMutation({
    mutationFn: () => {
      const n = Number(delta);
      if (!Number.isInteger(n) || n === 0) throw new Error("Escribe un entero distinto de cero (negativo para restar)");
      return adminApi.adjustStock(product.id, n, reason);
    },
    onSuccess: (res) => {
      queryClient.invalidateQueries({ queryKey: ["products"] });
      queryClient.invalidateQueries({ queryKey: ["admin"] });
      toast.success(`Stock actualizado: ${res.quantity} unidades`);
      onClose();
    },
    onError: (e) => setError(errorMessage(e)),
  });

  const submit = (e: FormEvent) => {
    e.preventDefault();
    setError(null);
    adjust.mutate();
  };

  return (
    <Modal open title={`Ajustar stock: ${product.name}`} onClose={onClose}>
      <p className="mb-4 text-sm text-slate-600">
        Stock actual: <strong>{product.available}</strong> unidades
      </p>
      <form onSubmit={submit} className="space-y-4">
        <TextField label="Cantidad a sumar o restar" type="number" value={delta} onChange={(e) => setDelta(e.target.value)} placeholder="Ej: 20 o -3" required />
        <SelectField label="Motivo" value={reason} onChange={(e) => setReason(e.target.value)}>
          {REASONS.map((r) => (
            <option key={r.value} value={r.value}>
              {r.label}
            </option>
          ))}
        </SelectField>
        {error && (
          <p role="alert" className="rounded-lg bg-red-50 p-3 text-sm text-red-700">
            {error}
          </p>
        )}
        <div className="flex justify-end gap-2">
          <Button variant="secondary" onClick={onClose}>
            Cancelar
          </Button>
          <Button type="submit" loading={adjust.isPending}>
            Aplicar
          </Button>
        </div>
      </form>

      <h3 className="mb-2 mt-6 text-sm font-semibold">Ultimos movimientos</h3>
      {movements.isLoading ? (
        <p className="text-sm text-slate-500">Cargando...</p>
      ) : movements.data && movements.data.length > 0 ? (
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
      ) : (
        <p className="text-sm text-slate-500">Sin movimientos.</p>
      )}
    </Modal>
  );
}

export default function InventoryPage() {
  const [page, setPage] = useState(0);
  const [search, setSearch] = useState("");
  const [q, setQ] = useState("");
  const [adjusting, setAdjusting] = useState<ProductListItem | null>(null);

  useEffect(() => {
    const id = setTimeout(() => {
      setQ(search);
      setPage(0);
    }, 350);
    return () => clearTimeout(id);
  }, [search]);

  const lowStock = useQuery({ queryKey: ["admin", "low-stock", 5], queryFn: () => adminApi.lowStock(5) });
  const query = useQuery({
    queryKey: ["products", "inventory", { q, page }],
    queryFn: () => catalogApi.list({ q, include_inactive: true, limit: PAGE_SIZE + 1, offset: page * PAGE_SIZE }),
    placeholderData: keepPreviousData,
  });
  const items = (query.data ?? []).filter((p) => p.kind === "simple").slice(0, PAGE_SIZE);

  return (
    <>
      <PageHeader title="Inventario" description="Existencias de productos simples. El stock de un combo depende de sus componentes." />

      {lowStock.data && lowStock.data.length > 0 && (
        <Card className="mb-6 border-amber-200 bg-amber-50 p-4">
          <p className="mb-2 flex items-center gap-2 text-sm font-semibold text-amber-900">
            <AlertTriangle className="h-4 w-4" aria-hidden /> {lowStock.data.length} producto(s) con 5 unidades o menos
          </p>
          <div className="flex flex-wrap gap-2">
            {lowStock.data.map((it) => (
              <Badge key={it.product_id} tone={it.quantity === 0 ? "red" : "amber"}>
                {it.name}: {it.quantity}
              </Badge>
            ))}
          </div>
        </Card>
      )}

      <input
        type="search"
        value={search}
        onChange={(e) => setSearch(e.target.value)}
        placeholder="Buscar por nombre o SKU"
        aria-label="Buscar productos"
        className="mb-4 h-10 w-full max-w-sm rounded-lg border border-slate-300 bg-white px-3 text-sm"
      />

      {query.isLoading ? (
        <PageLoader />
      ) : query.isError ? (
        <ErrorState message={errorMessage(query.error)} onRetry={() => query.refetch()} />
      ) : items.length === 0 ? (
        <EmptyState title="Sin productos" />
      ) : (
        <Table headers={[{ label: "Producto" }, { label: "SKU" }, { label: "Stock", className: "text-right" }, { label: "", className: "text-right" }]}>
          {items.map((p) => (
            <tr key={p.id}>
              <td className={`${td} font-medium`}>{p.name}</td>
              <td className={`${td} text-slate-500`}>{p.sku}</td>
              <td className={`${td} text-right font-semibold ${p.available === 0 ? "text-red-600" : p.available <= 5 ? "text-amber-600" : ""}`}>{p.available}</td>
              <td className={`${td} text-right`}>
                <Button variant="secondary" size="sm" onClick={() => setAdjusting(p)}>
                  Ajustar
                </Button>
              </td>
            </tr>
          ))}
        </Table>
      )}
      <Pagination page={page} hasNext={(query.data?.length ?? 0) > PAGE_SIZE} onChange={setPage} />

      {adjusting && <AdjustModal product={adjusting} onClose={() => setAdjusting(null)} />}
    </>
  );
}
