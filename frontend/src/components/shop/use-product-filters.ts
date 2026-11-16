"use client";

import { useEffect, useState } from "react";
import { usePathname, useRouter, useSearchParams } from "next/navigation";
import { useDebounced } from "@/lib/use-debounced";

/** Filtros del catalogo guardados en la URL (compartibles y con boton atras). */
export function useProductFilters() {
  const router = useRouter();
  const pathname = usePathname();
  const params = useSearchParams();

  const filters = {
    q: params.get("q") ?? "",
    category: params.get("category") ?? "",
    kind: params.get("kind") ?? "",
    inStock: params.get("stock") === "true",
    sort: params.get("sort") ?? "newest",
    page: Math.max(0, Number(params.get("page") ?? 0) || 0),
  };

  const [search, setSearch] = useState(filters.q);
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
    if (debouncedSearch !== filters.q) update({ q: debouncedSearch || null, page: null });
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [debouncedSearch]);

  const clear = () => {
    setSearch("");
    router.replace(pathname, { scroll: false });
  };
  const isFiltered = Boolean(filters.category || filters.kind || filters.inStock || filters.sort !== "newest" || filters.q);

  return { filters, search, setSearch, update, clear, isFiltered };
}

export type ProductFilters = ReturnType<typeof useProductFilters>;
