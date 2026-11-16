"use client";

import { useState } from "react";
import { keepPreviousData, useQuery } from "@tanstack/react-query";
import { catalogApi } from "@/lib/api/services";
import { useDebounced } from "@/lib/use-debounced";

const PAGE_SIZE = 15;

/** Listado paginado de productos (incluye inactivos) con busqueda con debounce. */
export function useAdminProductList(scope: string) {
  const [page, setPage] = useState(0);
  const [search, setSearchText] = useState("");
  const q = useDebounced(search);

  const query = useQuery({
    queryKey: ["products", scope, { q, page }],
    queryFn: () => catalogApi.list({ q, include_inactive: true, limit: PAGE_SIZE + 1, offset: page * PAGE_SIZE }),
    placeholderData: keepPreviousData,
  });

  const setSearch = (text: string) => {
    setSearchText(text);
    setPage(0);
  };

  return {
    query,
    items: query.data?.slice(0, PAGE_SIZE) ?? [],
    hasNext: (query.data?.length ?? 0) > PAGE_SIZE,
    page,
    setPage,
    search,
    setSearch,
  };
}
