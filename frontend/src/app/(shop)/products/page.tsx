"use client";

import { Suspense } from "react";
import { keepPreviousData, useQuery } from "@tanstack/react-query";
import { catalogApi } from "@/lib/api/services";
import { errorMessage } from "@/lib/api/client";
import { Pagination } from "@/components/ui/pagination";
import { EmptyState, ErrorState } from "@/components/ui/states";
import { ProductCard, ProductGridSkeleton } from "@/components/shop/product-card";
import { ProductFilterBar } from "@/components/shop/product-filters";
import { useProductFilters, type ProductFilters } from "@/components/shop/use-product-filters";

const PAGE_SIZE = 12;

function useProductPage({ q, category, kind, inStock, sort, page }: ProductFilters["filters"]) {
  return useQuery({
    queryKey: ["products", { q, category, kind, inStock, sort, page }],
    queryFn: () => catalogApi.list({ q, category_id: category, kind, in_stock: inStock, sort, limit: PAGE_SIZE + 1, offset: page * PAGE_SIZE }),
    placeholderData: keepPreviousData,
  });
}

function ProductGrid({ products }: { products: ReturnType<typeof useProductPage> }) {
  const items = products.data?.slice(0, PAGE_SIZE) ?? [];
  if (products.isLoading) return <ProductGridSkeleton count={PAGE_SIZE} />;
  if (products.isError) return <ErrorState message={errorMessage(products.error)} onRetry={() => products.refetch()} />;
  if (items.length === 0) return <EmptyState title="No encontramos productos" description="Prueba con otra busqueda o categoria." />;
  return (
    <div className={`grid grid-cols-1 gap-5 sm:grid-cols-2 lg:grid-cols-4 ${products.isPlaceholderData ? "opacity-60" : ""}`}>
      {items.map((p) => (
        <ProductCard key={p.id} product={p} />
      ))}
    </div>
  );
}

function ProductsView() {
  const state = useProductFilters();
  const products = useProductPage(state.filters);
  const hasNext = (products.data?.length ?? 0) > PAGE_SIZE;

  return (
    <div className="space-y-6">
      <h1 className="text-2xl font-bold">Productos</h1>
      <ProductFilterBar state={state} />
      <ProductGrid products={products} />
      <Pagination page={state.filters.page} hasNext={hasNext} onChange={(p) => state.update({ page: p > 0 ? String(p) : null })} />
    </div>
  );
}

export default function ProductsPage() {
  return (
    <Suspense fallback={<ProductGridSkeleton count={PAGE_SIZE} />}>
      <ProductsView />
    </Suspense>
  );
}
