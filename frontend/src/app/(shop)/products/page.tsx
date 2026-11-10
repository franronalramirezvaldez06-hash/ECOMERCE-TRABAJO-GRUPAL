"use client";

import { Suspense, useEffect, useState } from "react";
import { usePathname, useRouter, useSearchParams } from "next/navigation";
import { keepPreviousData, useQuery } from "@tanstack/react-query";
import { Search } from "lucide-react";
import { catalogApi } from "@/lib/api/services";
import { errorMessage } from "@/lib/api/client";
import { EmptyState, ErrorState, Pagination } from "@/components/ui/misc";
import { ProductCard, ProductGridSkeleton } from "@/components/shop/product-card";

const PAGE_SIZE = 12;

function useDebounced<T>(value: T, ms = 350): T {
  const [debounced, setDebounced] = useState(value);
  useEffect(() => {
    const id = setTimeout(() => setDebounced(value), ms);
    return () => clearTimeout(id);
  }, [value, ms]);
  return debounced;
}

function ProductsView() {
  const router = useRouter();
  const pathname = usePathname();
  const params = useSearchParams();

  const category = params.get("category") ?? "";
  const kind = params.get("kind") ?? "";
  const inStock = params.get("stock") === "true";
  const sort = params.get("sort") ?? "newest";
  const page = Math.max(0, Number(params.get("page") ?? 0) || 0);
  const urlQuery = params.get("q") ?? "";

  const [search, setSearch] = useState(urlQuery);
  const debouncedSearch = useDebounced(search);

  const update = (changes: Record<string, string | null>) => {
    const next = new URLSearchParams(params.toString());
    for (const [k, v] of Object.entries(changes)) {
      if (v) next.set(k, v);
      else next.delete(k);
    }
    router.replace(`${pathname}?${next.toString()}`, { scroll: false });
  };

  // Busqueda con debounce -> URL (reinicia la pagina).
  useEffect(() => {
    if (debouncedSearch !== urlQuery) update({ q: debouncedSearch || null, page: null });
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [debouncedSearch]);

  const categories = useQuery({ queryKey: ["categories"], queryFn: catalogApi.categories });
  const products = useQuery({
    queryKey: ["products", { q: urlQuery, category, kind, inStock, sort, page }],
    queryFn: () =>
      catalogApi.list({ q: urlQuery, category_id: category, kind, in_stock: inStock, sort, limit: PAGE_SIZE + 1, offset: page * PAGE_SIZE }),
    placeholderData: keepPreviousData,
  });

  const items = products.data?.slice(0, PAGE_SIZE) ?? [];
  const hasNext = (products.data?.length ?? 0) > PAGE_SIZE;

  return (
    <div className="space-y-6">
      <h1 className="text-2xl font-bold">Productos</h1>

      <div className="flex flex-col gap-3 sm:flex-row">
        <div className="relative flex-1">
          <Search className="pointer-events-none absolute left-3 top-1/2 h-4 w-4 -translate-y-1/2 text-slate-400" aria-hidden />
          <input
            type="search"
            value={search}
            onChange={(e) => setSearch(e.target.value)}
            placeholder="Buscar por nombre o SKU"
            aria-label="Buscar productos"
            className="h-10 w-full rounded-lg border border-slate-300 bg-white pl-9 pr-3 text-sm"
          />
        </div>
        <select
          value={category}
          onChange={(e) => update({ category: e.target.value || null, page: null })}
          aria-label="Filtrar por categoria"
          className="h-10 rounded-lg border border-slate-300 bg-white px-3 text-sm sm:w-56"
        >
          <option value="">Todas las categorias</option>
          {categories.data?.map((c) => (
            <option key={c.id} value={c.id}>
              {c.name}
            </option>
          ))}
        </select>
      </div>

      <div className="flex flex-wrap items-center gap-3" aria-label="Filtros de productos">
        <select value={kind} onChange={(e) => update({ kind: e.target.value || null, page: null })}
          aria-label="Filtrar por tipo" className="h-10 rounded-lg border border-slate-300 bg-white px-3 text-sm">
          <option value="">Todos los tipos</option>
          <option value="simple">Productos individuales</option>
          <option value="bundle">Combos</option>
        </select>
        <select value={sort} onChange={(e) => update({ sort: e.target.value === "newest" ? null : e.target.value, page: null })}
          aria-label="Ordenar productos" className="h-10 rounded-lg border border-slate-300 bg-white px-3 text-sm">
          <option value="newest">Más recientes</option>
          <option value="name">Nombre A–Z</option>
        </select>
        <label className="flex items-center gap-2 text-sm text-slate-700">
          <input type="checkbox" checked={inStock} onChange={(e) => update({ stock: e.target.checked ? "true" : null, page: null })} />
          Solo disponibles
        </label>
        {(category || kind || inStock || sort !== "newest" || urlQuery) && (
          <button type="button" className="text-sm font-medium text-brand-700" onClick={() => { setSearch(""); router.replace(pathname, { scroll: false }); }}>
            Limpiar filtros
          </button>
        )}
      </div>

      {products.isLoading ? (
        <ProductGridSkeleton count={PAGE_SIZE} />
      ) : products.isError ? (
        <ErrorState message={errorMessage(products.error)} onRetry={() => products.refetch()} />
      ) : items.length === 0 ? (
        <EmptyState title="No encontramos productos" description="Prueba con otra busqueda o categoria." />
      ) : (
        <div className={`grid grid-cols-1 gap-5 sm:grid-cols-2 lg:grid-cols-4 ${products.isPlaceholderData ? "opacity-60" : ""}`}>
          {items.map((p) => (
            <ProductCard key={p.id} product={p} />
          ))}
        </div>
      )}

      <Pagination page={page} hasNext={hasNext} onChange={(p) => update({ page: p > 0 ? String(p) : null })} />
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
