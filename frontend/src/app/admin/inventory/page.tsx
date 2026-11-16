"use client";

import { useState } from "react";
import { useQuery } from "@tanstack/react-query";
import { AlertTriangle } from "lucide-react";
import { adminApi } from "@/lib/api/services";
import type { LowStockItem, ProductListItem } from "@/lib/api/types";
import { Button } from "@/components/ui/button";
import { Badge } from "@/components/ui/badge";
import { Card } from "@/components/ui/card";
import { Pagination } from "@/components/ui/pagination";
import { EmptyState } from "@/components/ui/states";
import { PageHeader, QueryView, SearchInput, Table, td } from "@/components/admin/admin-ui";
import { AdjustModal } from "@/components/admin/inventory/adjust-modal";
import { useAdminProductList } from "@/components/admin/use-admin-product-list";

const HEADERS = [{ label: "Producto" }, { label: "SKU" }, { label: "Stock", className: "text-right" }, { label: "", className: "text-right" }];

function LowStockAlert({ items }: { items: LowStockItem[] | undefined }) {
  if (!items || items.length === 0) return null;
  return (
    <Card className="mb-6 border-amber-200 bg-amber-50 p-4">
      <p className="mb-2 flex items-center gap-2 text-sm font-semibold text-amber-900">
        <AlertTriangle className="h-4 w-4" aria-hidden /> {items.length} producto(s) con 5 unidades o menos
      </p>
      <div className="flex flex-wrap gap-2">
        {items.map((it) => (
          <Badge key={it.product_id} tone={it.quantity === 0 ? "red" : "amber"}>
            {it.name}: {it.quantity}
          </Badge>
        ))}
      </div>
    </Card>
  );
}

const stockTone = (available: number) => (available === 0 ? "text-red-600" : available <= 5 ? "text-amber-600" : "");

function StockRow({ product: p, onAdjust }: { product: ProductListItem; onAdjust: () => void }) {
  return (
    <tr>
      <td className={`${td} font-medium`}>{p.name}</td>
      <td className={`${td} text-slate-500`}>{p.sku}</td>
      <td className={`${td} text-right font-semibold ${stockTone(p.available)}`}>{p.available}</td>
      <td className={`${td} text-right`}>
        <Button variant="secondary" size="sm" onClick={onAdjust}>
          Ajustar
        </Button>
      </td>
    </tr>
  );
}

export default function InventoryPage() {
  const list = useAdminProductList("inventory");
  const lowStock = useQuery({ queryKey: ["admin", "low-stock", 5], queryFn: () => adminApi.lowStock(5) });
  const [adjusting, setAdjusting] = useState<ProductListItem | null>(null);
  const items = list.items.filter((p) => p.kind === "simple");

  return (
    <>
      <PageHeader title="Inventario" description="Existencias de productos simples. El stock de un combo depende de sus componentes." />
      <LowStockAlert items={lowStock.data} />
      <SearchInput value={list.search} onChange={list.setSearch} />
      <QueryView query={list.query} empty={items.length === 0 && <EmptyState title="Sin productos" />}>
        <Table headers={HEADERS}>
          {items.map((p) => (
            <StockRow key={p.id} product={p} onAdjust={() => setAdjusting(p)} />
          ))}
        </Table>
      </QueryView>
      <Pagination page={list.page} hasNext={list.hasNext} onChange={list.setPage} />
      {adjusting && <AdjustModal product={adjusting} onClose={() => setAdjusting(null)} />}
    </>
  );
}
