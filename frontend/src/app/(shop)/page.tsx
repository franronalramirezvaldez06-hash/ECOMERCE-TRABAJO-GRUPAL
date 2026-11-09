"use client";

import Link from "next/link";
import { useQuery } from "@tanstack/react-query";
import { PackageCheck, ShieldCheck, Truck } from "lucide-react";
import { catalogApi } from "@/lib/api/services";
import { errorMessage } from "@/lib/api/client";
import { LinkButton } from "@/components/ui/button";
import { ErrorState } from "@/components/ui/misc";
import { ProductCard, ProductGridSkeleton } from "@/components/shop/product-card";

export default function HomePage() {
  const products = useQuery({ queryKey: ["products", "featured"], queryFn: () => catalogApi.list({ limit: 8 }) });
  const categories = useQuery({ queryKey: ["categories"], queryFn: catalogApi.categories });

  return (
    <div className="space-y-12">
      <section className="overflow-hidden rounded-2xl bg-gradient-to-br from-brand-600 to-violet-700 px-6 py-14 text-white sm:px-12">
        <h1 className="max-w-2xl text-3xl font-bold leading-tight sm:text-5xl">Todo lo que necesitas, en un solo lugar</h1>
        <p className="mt-4 max-w-xl text-brand-100">
          Productos y combos con descuento, pagos seguros con varias pasarelas e inventario actualizado en tiempo real.
        </p>
        <LinkButton href="/products" size="lg" variant="secondary" className="mt-8">
          Ver productos
        </LinkButton>
      </section>

      <section className="grid gap-4 sm:grid-cols-3" aria-label="Beneficios">
        {[
          { icon: PackageCheck, title: "Combos con descuento", text: "Arma paquetes y ahorra en cada compra." },
          { icon: ShieldCheck, title: "Pago seguro", text: "Stripe, PayPal y MercadoPago con reintentos automaticos." },
          { icon: Truck, title: "Seguimiento", text: "Consulta el estado de tus pedidos y cancela si hace falta." },
        ].map(({ icon: Icon, title, text }) => (
          <div key={title} className="flex gap-3 rounded-xl border border-slate-200 bg-white p-5">
            <Icon className="h-6 w-6 shrink-0 text-brand-600" aria-hidden />
            <div>
              <h2 className="font-semibold">{title}</h2>
              <p className="text-sm text-slate-500">{text}</p>
            </div>
          </div>
        ))}
      </section>

      {categories.data && categories.data.length > 0 && (
        <section aria-label="Categorias">
          <h2 className="mb-4 text-xl font-semibold">Categorias</h2>
          <div className="flex flex-wrap gap-2">
            {categories.data.map((c) => (
              <Link
                key={c.id}
                href={`/products?category=${c.id}`}
                className="rounded-full border border-slate-300 bg-white px-4 py-1.5 text-sm font-medium text-slate-700 hover:border-brand-500 hover:text-brand-700"
              >
                {c.name}
              </Link>
            ))}
          </div>
        </section>
      )}

      <section aria-label="Productos destacados">
        <div className="mb-4 flex items-end justify-between">
          <h2 className="text-xl font-semibold">Novedades</h2>
          <Link href="/products" className="text-sm font-medium text-brand-700 hover:underline">
            Ver todo
          </Link>
        </div>
        {products.isLoading ? (
          <ProductGridSkeleton />
        ) : products.isError ? (
          <ErrorState message={errorMessage(products.error)} onRetry={() => products.refetch()} />
        ) : products.data && products.data.length > 0 ? (
          <div className="grid grid-cols-1 gap-5 sm:grid-cols-2 lg:grid-cols-4">
            {products.data.map((p) => (
              <ProductCard key={p.id} product={p} />
            ))}
          </div>
        ) : (
          <p className="rounded-xl border border-dashed border-slate-300 bg-white p-10 text-center text-slate-500">
            Aun no hay productos. Un administrador puede crearlos desde el panel.
          </p>
        )}
      </section>
    </div>
  );
}
