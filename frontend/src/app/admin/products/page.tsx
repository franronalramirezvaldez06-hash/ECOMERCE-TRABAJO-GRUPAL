"use client";

import { useEffect, useState } from "react";
import { keepPreviousData, useMutation, useQuery, useQueryClient } from "@tanstack/react-query";
import { PackagePlus, Pencil, Plus } from "lucide-react";
import { adminApi, catalogApi } from "@/lib/api/services";
import { errorMessage } from "@/lib/api/client";
import type { Product } from "@/lib/api/types";
import { formatMoney } from "@/lib/format";
import { useToast } from "@/providers/toast-provider";
import { Button } from "@/components/ui/button";
import { Badge, EmptyState, ErrorState, PageLoader, Pagination } from "@/components/ui/misc";
import { PageHeader, Table, td } from "@/components/admin/admin-ui";
import { BundleModal, ProductModal } from "@/components/admin/product-forms";

const PAGE_SIZE = 15;

export default function AdminProductsPage() {
  const toast = useToast();
  const queryClient = useQueryClient();
  const [page, setPage] = useState(0);
  const [search, setSearch] = useState("");
  const [q, setQ] = useState("");
  const [editing, setEditing] = useState<Product | null>(null);
  const [creating, setCreating] = useState(false);
  const [creatingBundle, setCreatingBundle] = useState(false);

  useEffect(() => {
    const id = setTimeout(() => {
      setQ(search);
      setPage(0);
    }, 350);
    return () => clearTimeout(id);
  }, [search]);

  const query = useQuery({
    queryKey: ["products", "admin", { q, page }],
    queryFn: () => catalogApi.list({ q, include_inactive: true, limit: PAGE_SIZE + 1, offset: page * PAGE_SIZE }),
    placeholderData: keepPreviousData,
  });

  const toggle = useMutation({
    mutationFn: (p: Product) => adminApi.updateProduct(p.id, { active: !p.active }),
    onSuccess: (p) => {
      queryClient.invalidateQueries({ queryKey: ["products"] });
      toast.success(p.active ? "Producto activado" : "Producto desactivado");
    },
    onError: (e) => toast.error(errorMessage(e)),
  });

  const items = query.data?.slice(0, PAGE_SIZE) ?? [];

  return (
    <>
      <PageHeader
        title="Productos"
        description="Productos simples y combos. Los productos inactivos no aparecen en la tienda."
        actions={
          <>
            <Button variant="secondary" onClick={() => setCreatingBundle(true)}>
              <PackagePlus className="h-4 w-4" aria-hidden /> Nuevo combo
            </Button>
            <Button onClick={() => setCreating(true)}>
              <Plus className="h-4 w-4" aria-hidden /> Nuevo producto
            </Button>
          </>
        }
      />

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
        <EmptyState title="Sin productos" description="Crea tu primer producto para empezar a vender." />
      ) : (
        <Table
          headers={[
            { label: "Producto" },
            { label: "Tipo" },
            { label: "Precio", className: "text-right" },
            { label: "Disponible", className: "text-right" },
            { label: "Estado" },
            { label: "", className: "text-right" },
          ]}
        >
          {items.map((p) => (
            <tr key={p.id} className={p.active ? "" : "bg-slate-50 text-slate-500"}>
              <td className={td}>
                <p className="font-medium text-slate-900">{p.name}</p>
                <p className="text-xs text-slate-500">{p.sku}</p>
              </td>
              <td className={td}>{p.kind === "bundle" ? <Badge tone="purple">Combo</Badge> : <Badge>Simple</Badge>}</td>
              <td className={`${td} text-right`}>{formatMoney(p.price_cents_total)}</td>
              <td className={`${td} text-right ${p.available === 0 ? "font-semibold text-red-600" : ""}`}>{p.available}</td>
              <td className={td}>{p.active ? <Badge tone="green">Activo</Badge> : <Badge tone="amber">Inactivo</Badge>}</td>
              <td className={`${td} whitespace-nowrap text-right`}>
                <Button variant="ghost" size="sm" onClick={() => setEditing(p)} aria-label={`Editar ${p.name}`}>
                  <Pencil className="h-4 w-4" aria-hidden /> Editar
                </Button>
                <Button variant="ghost" size="sm" onClick={() => toggle.mutate(p)} disabled={toggle.isPending}>
                  {p.active ? "Desactivar" : "Activar"}
                </Button>
              </td>
            </tr>
          ))}
        </Table>
      )}
      <Pagination page={page} hasNext={(query.data?.length ?? 0) > PAGE_SIZE} onChange={setPage} />

      {creating && <ProductModal open product={null} onClose={() => setCreating(false)} />}
      {editing && <ProductModal open product={editing} onClose={() => setEditing(null)} />}
      {creatingBundle && <BundleModal open onClose={() => setCreatingBundle(false)} />}
    </>
  );
}
