"use client";

import { useEffect, useState } from "react";

/** Devuelve value con retraso: solo cambia cuando deja de cambiar durante ms. */
export function useDebounced<T>(value: T, ms = 350): T {
  const [debounced, setDebounced] = useState(value);
  useEffect(() => {
    const id = setTimeout(() => setDebounced(value), ms);
    return () => clearTimeout(id);
  }, [value, ms]);
  return debounced;
}
