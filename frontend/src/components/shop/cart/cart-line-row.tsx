"use client";

import Link from "next/link";
import { Trash2 } from "lucide-react";
import type { CartLine } from "@/lib/cart/cart-store";
import { formatMoney } from "@/lib/format";
import { useCart } from "@/providers/cart-provider";
import { ProductImage } from "@/components/shop/product-image";
import { QuantityStepper } from "@/components/shop/quantity-stepper";

export function CartLineRow({ line }: { line: CartLine }) {
  const { dispatch } = useCart();
  return (
    <div className="flex gap-4 p-4">
      <ProductImage src={line.imageUrl} alt={line.name} className="h-20 w-20 shrink-0 rounded-lg" />
      <div className="flex flex-1 flex-col justify-between gap-2 sm:flex-row sm:items-center">
        <div>
          <Link href={`/products/${line.productId}`} className="font-medium hover:text-brand-700">
            {line.name}
          </Link>
          <p className="text-sm text-slate-500">{formatMoney(line.unitPriceCents)} c/u</p>
        </div>
        <div className="flex items-center gap-4">
          <QuantityStepper
            value={line.quantity}
            max={line.maxQuantity}
            onChange={(q) => dispatch({ type: "setQuantity", productId: line.productId, quantity: q })}
            label={`Cantidad de ${line.name}`}
          />
          <span className="w-24 text-right font-semibold">{formatMoney(line.unitPriceCents * line.quantity)}</span>
          <button
            onClick={() => dispatch({ type: "remove", productId: line.productId })}
            aria-label={`Quitar ${line.name}`}
            className="rounded-md p-2 text-slate-400 hover:bg-red-50 hover:text-red-600"
          >
            <Trash2 className="h-4 w-4" />
          </button>
        </div>
      </div>
    </div>
  );
}
