"use client";

import { useState } from "react";
import { useMutation, useQueryClient } from "@tanstack/react-query";
import { PackagePlus, Plus } from "lucide-react";
import { adminApi } from "@/lib/api/services";
import { errorMessage } from "@/lib/api/client";
import type { Product } from "@/lib/api/types";
import { useToast } from "@/providers/toast-provider";
import { Button } from "@/components/ui/button";
import { Pagination } from "@/components/ui/pagination";
import { EmptyState } from "@/components/ui/states";
import { PageHeader, QueryView, SearchInput } from "@/components/admin/admin-ui";
import { BundleModal } from "@/components/admin/products/bundle-modal";
import { ProductModal } from "@/components/admin/products/product-modal";
import { ProductsTable } from "@/components/admin/products/products-table";
import { useAdminProductList } from "@/components/admin/use-admin-product-list";

type Dialog = { kind: "product"; product: Product | null } | { kind: "bundle" } | null;

function useToggleActive() {
  const toast = useToast();
  const queryClient = useQueryClient();
  return useMutation({
    mutationFn: (p: Product) => adminApi.updateProduct(p.id, { active: !p.active }),
    onSuccess: (p) => {
      queryClient.invalidateQueries({ queryKey: ["products"] });
      toast.success(p.active ? "Producto activado" : "Producto desactivado");
    },
    onError: (e) => toast.error(errorMessage(e)),
  });
}

export default function AdminProductsPage() {
  const list = useAdminProductList("admin");
  const toggle = useToggleActive();
  const [dialog, setDialog] = useState<Dialog>(null);
  const close = () => setDialog(null);

  const actions = (
    <>
      <Button variant="secondary" onClick={() => setDialog({ kind: "bundle" })}>
        <PackagePlus className="h-4 w-4" aria-hidden /> Nuevo combo
      </Button>
      <Button onClick={() => setDialog({ kind: "product", product: null })}>
        <Plus className="h-4 w-4" aria-hidden /> Nuevo producto
      </Button>
    </>
  );

  return (
    <>
      <PageHeader title="Productos" description="Productos simples y combos. Los productos inactivos no aparecen en la tienda." actions={actions} />
      <SearchInput value={list.search} onChange={list.setSearch} />
      <QueryView
        query={list.query}
        empty={list.items.length === 0 && <EmptyState title="Sin productos" description="Crea tu primer producto para empezar a vender." />}
      >
        <ProductsTable
          items={list.items}
          onEdit={(product) => setDialog({ kind: "product", product })}
          onToggle={(product) => toggle.mutate(product)}
          toggling={toggle.isPending}
        />
      </QueryView>
      <Pagination page={list.page} hasNext={list.hasNext} onChange={list.setPage} />

      {dialog?.kind === "product" && <ProductModal open product={dialog.product} onClose={close} />}
      {dialog?.kind === "bundle" && <BundleModal open onClose={close} />}
    </>
  );
}
