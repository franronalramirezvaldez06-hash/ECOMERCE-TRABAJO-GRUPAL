"use client";

import Link from "next/link";
import { ShoppingCart } from "lucide-react";
import type { ProductListItem } from "@/lib/api/types";
import { formatMoney } from "@/lib/format";
import { useCart } from "@/providers/cart-provider";
import { useToast } from "@/providers/toast-provider";
import { Badge } from "@/components/ui/misc";
import { Button } from "@/components/ui/button";
import { ProductImage } from "./product-image";

export function ProductCard({ product }: { product: ProductListItem }) {
  const { dispatch } = useCart();
  const toast = useToast();
  const soldOut = product.available <= 0;

  const add = () => {
    dispatch({
      type: "add",
      quantity: 1,
      line: {
        productId: product.id,
        name: product.name,
        unitPriceCents: product.price_cents_total,
        imageUrl: product.image_url,
        maxQuantity: product.available,
      },
    });
    toast.success(`${product.name} agregado al carrito`);
  };

  return (
    <article className="group flex flex-col overflow-hidden rounded-xl border border-slate-200 bg-white shadow-sm transition-shadow hover:shadow-md">
      <Link href={`/products/${product.id}`} className="relative block">
        <ProductImage src={product.image_url} alt={product.name} className="aspect-square w-full" />
        <div className="absolute left-3 top-3 flex gap-1.5">
          {product.kind === "bundle" && <Badge tone="purple">Combo{product.bundle_discount_pct > 0 ? ` -${product.bundle_discount_pct}%` : ""}</Badge>}
          {soldOut && <Badge tone="red">Agotado</Badge>}
        </div>
      </Link>
      <div className="flex flex-1 flex-col p-4">
        <Link href={`/products/${product.id}`} className="line-clamp-2 font-medium text-slate-900 group-hover:text-brand-700">
          {product.name}
        </Link>
        <p className="mt-1 text-xs text-slate-500">SKU {product.sku}</p>
        <div className="mt-auto flex items-center justify-between pt-4">
          <span className="text-lg font-semibold">{formatMoney(product.price_cents_total)}</span>
          <Button size="sm" onClick={add} disabled={soldOut} aria-label={`Agregar ${product.name} al carrito`}>
            <ShoppingCart className="h-4 w-4" aria-hidden />
            Agregar
          </Button>
        </div>
        {!soldOut && product.available <= 5 && (
          <p className="mt-2 text-xs font-medium text-amber-600">Quedan solo {product.available}</p>
        )}
      </div>
    </article>
  );
}

export function ProductGridSkeleton({ count = 8 }: { count?: number }) {
  return (
    <div className="grid grid-cols-1 gap-5 sm:grid-cols-2 lg:grid-cols-4">
      {Array.from({ length: count }).map((_, i) => (
        <div key={i} className="animate-pulse overflow-hidden rounded-xl border border-slate-200 bg-white">
          <div className="aspect-square bg-slate-100" />
          <div className="space-y-3 p-4">
            <div className="h-4 w-3/4 rounded bg-slate-100" />
            <div className="h-3 w-1/3 rounded bg-slate-100" />
            <div className="h-8 w-full rounded bg-slate-100" />
          </div>
        </div>
      ))}
    </div>
  );
}
