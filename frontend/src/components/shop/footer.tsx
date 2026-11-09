import Link from "next/link";

export function Footer() {
  return (
    <footer className="mt-16 border-t border-slate-200 bg-white">
      <div className="mx-auto flex max-w-7xl flex-col items-center justify-between gap-3 px-4 py-8 text-sm text-slate-500 sm:flex-row sm:px-6">
        <p>Sistema ecommerce &mdash; ventas, inventarios y pasarelas de pago &middot; Grupo 3</p>
        <nav className="flex gap-4" aria-label="Pie de pagina">
          <Link href="/products" className="hover:text-slate-900">
            Productos
          </Link>
          <Link href="/orders" className="hover:text-slate-900">
            Mis pedidos
          </Link>
        </nav>
      </div>
    </footer>
  );
}
