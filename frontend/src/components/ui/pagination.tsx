"use client";

import { Button } from "./button";

export function Pagination({
  page,
  hasNext,
  onChange,
}: {
  page: number;
  hasNext: boolean;
  onChange: (page: number) => void;
}) {
  if (page === 0 && !hasNext) return null;
  return (
    <nav aria-label="Paginacion" className="flex items-center justify-center gap-3 pt-6">
      <Button variant="secondary" size="sm" disabled={page === 0} onClick={() => onChange(page - 1)}>
        Anterior
      </Button>
      <span className="text-sm text-slate-600">Pagina {page + 1}</span>
      <Button variant="secondary" size="sm" disabled={!hasNext} onClick={() => onChange(page + 1)}>
        Siguiente
      </Button>
    </nav>
  );
}
