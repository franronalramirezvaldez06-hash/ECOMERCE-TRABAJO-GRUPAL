"use client";

import { createContext, useContext, useEffect, useMemo, useReducer, useState, type Dispatch, type ReactNode } from "react";
import {
  cartCount,
  cartReducer,
  cartSubtotalCents,
  EMPTY_CART,
  parseStoredCart,
  type CartAction,
  type CartState,
} from "@/lib/cart/cart-store";

const STORAGE_KEY = "ecommerce.cart";

interface CartContextValue {
  state: CartState;
  dispatch: Dispatch<CartAction>;
  count: number;
  subtotalCents: number;
  /** false hasta leer localStorage (evita parpadeos e inconsistencias de hidratacion). */
  ready: boolean;
}

const CartContext = createContext<CartContextValue | null>(null);

export function CartProvider({ children }: { children: ReactNode }) {
  const [state, dispatch] = useReducer(cartReducer, EMPTY_CART);
  const [ready, setReady] = useState(false);

  useEffect(() => {
    try {
      dispatch({ type: "hydrate", state: parseStoredCart(window.localStorage.getItem(STORAGE_KEY)) });
    } catch {
      /* sin almacenamiento */
    }
    setReady(true);
  }, []);

  useEffect(() => {
    if (!ready) return;
    try {
      window.localStorage.setItem(STORAGE_KEY, JSON.stringify(state));
    } catch {
      /* sin almacenamiento */
    }
  }, [state, ready]);

  // Sincroniza el carrito entre pestanas.
  useEffect(() => {
    const onStorage = (e: StorageEvent) => {
      if (e.key === STORAGE_KEY) dispatch({ type: "hydrate", state: parseStoredCart(e.newValue) });
    };
    window.addEventListener("storage", onStorage);
    return () => window.removeEventListener("storage", onStorage);
  }, []);

  const value = useMemo<CartContextValue>(
    () => ({ state, dispatch, count: cartCount(state), subtotalCents: cartSubtotalCents(state), ready }),
    [state, ready],
  );
  return <CartContext.Provider value={value}>{children}</CartContext.Provider>;
}

export function useCart(): CartContextValue {
  const ctx = useContext(CartContext);
  if (!ctx) throw new Error("useCart debe usarse dentro de CartProvider");
  return ctx;
}
