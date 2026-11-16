"use client";

import { useQuery } from "@tanstack/react-query";
import { Search } from "lucide-react";
import { catalogApi } from "@/lib/api/services";
import type { ProductFilters } from "./use-product-filters";

const SELECT = "h-10 rounded-lg border border-slate-300 bg-white px-3 text-sm";

function SearchAndCategory({ state }: { state: ProductFilters }) {
  const categories = useQuery({ queryKey: ["categories"], queryFn: catalogApi.categories });
  return (
    <div className="flex flex-col gap-3 sm:flex-row">
      <div className="relative flex-1">
        <Search className="pointer-events-none absolute left-3 top-1/2 h-4 w-4 -translate-y-1/2 text-slate-400" aria-hidden />
        <input
          type="search"
          value={state.search}
          onChange={(e) => state.setSearch(e.target.value)}
          placeholder="Buscar por nombre o SKU"
          aria-label="Buscar productos"
          className="h-10 w-full rounded-lg border border-slate-300 bg-white pl-9 pr-3 text-sm"
        />
      </div>
      <select
        value={state.filters.category}
        onChange={(e) => state.update({ category: e.target.value || null, page: null })}
        aria-label="Filtrar por categoria"
        className={`${SELECT} sm:w-56`}
      >
        <option value="">Todas las categorias</option>
        {categories.data?.map((c) => (
          <option key={c.id} value={c.id}>
            {c.name}
          </option>
        ))}
      </select>
    </div>
  );
}

function KindSortAndStock({ state }: { state: ProductFilters }) {
  const { filters, update } = state;
  return (
    <div className="flex flex-wrap items-center gap-3" aria-label="Filtros de productos">
      <select value={filters.kind} onChange={(e) => update({ kind: e.target.value || null, page: null })} aria-label="Filtrar por tipo" className={SELECT}>
        <option value="">Todos los tipos</option>
        <option value="simple">Productos individuales</option>
        <option value="bundle">Combos</option>
      </select>
      <select
        value={filters.sort}
        onChange={(e) => update({ sort: e.target.value === "newest" ? null : e.target.value, page: null })}
        aria-label="Ordenar productos"
        className={SELECT}
      >
        <option value="newest">Más recientes</option>
        <option value="name">Nombre A–Z</option>
      </select>
      <label className="flex items-center gap-2 text-sm text-slate-700">
        <input type="checkbox" checked={filters.inStock} onChange={(e) => update({ stock: e.target.checked ? "true" : null, page: null })} />
        Solo disponibles
      </label>
      {state.isFiltered && (
        <button type="button" className="text-sm font-medium text-brand-700" onClick={state.clear}>
          Limpiar filtros
        </button>
      )}
    </div>
  );
}

export function ProductFilterBar({ state }: { state: ProductFilters }) {
  return (
    <>
      <SearchAndCategory state={state} />
      <KindSortAndStock state={state} />
    </>
  );
}
