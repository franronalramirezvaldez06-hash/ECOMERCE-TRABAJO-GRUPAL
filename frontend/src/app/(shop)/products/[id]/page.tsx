"use client";

import { useState } from "react";
import Link from "next/link";
import { useParams } from "next/navigation";
import { useQuery } from "@tanstack/react-query";
import { ChevronLeft, ShoppingCart } from "lucide-react";
import { catalogApi } from "@/lib/api/services";
import { ApiError, errorMessage } from "@/lib/api/client";
import type { ProductDetail } from "@/lib/api/types";
import { formatMoney } from "@/lib/format";
import { useCart } from "@/providers/cart-provider";
import { useToast } from "@/providers/toast-provider";
import { Button, LinkButton } from "@/components/ui/button";
import { Badge } from "@/components/ui/badge";
import { Card } from "@/components/ui/card";
import { PageLoader } from "@/components/ui/spinner";
import { EmptyState, ErrorState } from "@/components/ui/states";
import { ProductImage } from "@/components/shop/product-image";
import { QuantityStepper } from "@/components/shop/quantity-stepper";
import { BundleTree } from "@/components/shop/bundle-tree";

function ProductBadges({ product }: { product: ProductDetail }) {
  const discount = product.bundle_discount_pct > 0 ? ` -${product.bundle_discount_pct}%` : "";
  return (
    <div className="flex flex-wrap items-center gap-2">
      {product.kind === "bundle" && <Badge tone="purple">Combo{discount}</Badge>}
      {!product.active && <Badge tone="amber">Inactivo</Badge>}
      {product.available <= 0 ? <Badge tone="red">Agotado</Badge> : <Badge tone="green">{product.available} disponibles</Badge>}
    </div>
  );
}

function AddToCart({ product }: { product: ProductDetail }) {
  const { dispatch } = useCart();
  const toast = useToast();
  const [quantity, setQuantity] = useState(1);

  const add = () => {
    const line = {
      productId: product.id,
      name: product.name,
      unitPriceCents: product.price_cents_total,
      imageUrl: product.image_url,
      maxQuantity: product.available,
    };
    dispatch({ type: "add", quantity, line });
    toast.success(`${quantity} x ${product.name} agregado al carrito`);
  };

  return (
    <div className="flex flex-wrap items-center gap-3 pt-2">
      <QuantityStepper value={quantity} max={Math.max(1, product.available)} onChange={setQuantity} />
      <Button size="lg" onClick={add} disabled={product.available <= 0 || !product.active}>
        <ShoppingCart className="h-5 w-5" aria-hidden />
        Agregar al carrito
      </Button>
    </div>
  );
}

function BundleContents({ product }: { product: ProductDetail }) {
  if (product.kind !== "bundle" || !product.tree.children) return null;
  return (
    <Card className="p-5">
      <h2 className="mb-3 font-semibold">Este combo incluye</h2>
      <BundleTree nodes={product.tree.children} />
      {product.bundle_discount_pct > 0 && (
        <p className="mt-3 border-t border-slate-100 pt-3 text-sm text-emerald-700">Ahorras {product.bundle_discount_pct}% comprando el combo completo.</p>
      )}
    </Card>
  );
}

function ProductView({ product }: { product: ProductDetail }) {
  return (
    <div className="space-y-6">
      <Link href="/products" className="inline-flex items-center gap-1 text-sm text-slate-600 hover:text-slate-900">
        <ChevronLeft className="h-4 w-4" aria-hidden /> Volver a productos
      </Link>
      <div className="grid gap-8 lg:grid-cols-2">
        <ProductImage src={product.image_url} alt={product.name} className="aspect-square w-full rounded-2xl border border-slate-200" />
        <div className="space-y-5">
          <ProductBadges product={product} />
          <h1 className="text-3xl font-bold">{product.name}</h1>
          <p className="text-3xl font-semibold text-brand-700">{formatMoney(product.price_cents_total)}</p>
          {product.description && <p className="whitespace-pre-line text-slate-600">{product.description}</p>}
          <p className="text-xs text-slate-500">SKU {product.sku}</p>
          <AddToCart product={product} />
          <BundleContents product={product} />
        </div>
      </div>
    </div>
  );
}

export default function ProductDetailPage() {
  const { id } = useParams<{ id: string }>();
  const query = useQuery({ queryKey: ["product", id], queryFn: () => catalogApi.get(id) });

  if (query.isLoading) return <PageLoader />;
  if (query.error instanceof ApiError && query.error.status === 404) {
    return <EmptyState title="Producto no encontrado" description="Puede que ya no este disponible." action={<LinkButton href="/products">Ver productos</LinkButton>} />;
  }
  if (query.isError) return <ErrorState message={errorMessage(query.error)} onRetry={() => query.refetch()} />;
  return query.data ? <ProductView product={query.data} /> : null;
}
