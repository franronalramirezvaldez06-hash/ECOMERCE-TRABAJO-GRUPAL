"use client";

import { useEffect, useState, type FormEvent } from "react";
import { Tag } from "lucide-react";
import { useCart } from "@/providers/cart-provider";
import { Button } from "@/components/ui/button";

export function CouponForm() {
  const { state, dispatch } = useCart();
  const [coupon, setCoupon] = useState("");

  // Refleja el cupon guardado (p. ej. al hidratar desde localStorage o quitarlo).
  useEffect(() => setCoupon(state.coupon), [state.coupon]);

  const apply = (e: FormEvent) => {
    e.preventDefault();
    dispatch({ type: "setCoupon", coupon });
  };
  const clear = () => {
    setCoupon("");
    dispatch({ type: "setCoupon", coupon: "" });
  };

  return (
    <>
      <form className="flex gap-2" onSubmit={apply}>
        <div className="relative flex-1">
          <Tag className="pointer-events-none absolute left-3 top-1/2 h-4 w-4 -translate-y-1/2 text-slate-400" aria-hidden />
          <input
            value={coupon}
            onChange={(e) => setCoupon(e.target.value)}
            placeholder="Cupon de descuento"
            aria-label="Cupon de descuento"
            className="h-10 w-full rounded-lg border border-slate-300 pl-9 pr-3 text-sm uppercase"
          />
        </div>
        <Button type="submit" variant="secondary">
          Aplicar
        </Button>
      </form>
      {state.coupon && (
        <p className="-mt-2 text-xs text-slate-500">
          Cupon aplicado: <strong>{state.coupon}</strong>{" "}
          <button className="text-brand-700 underline" onClick={clear}>
            quitar
          </button>
        </p>
      )}
    </>
  );
}
