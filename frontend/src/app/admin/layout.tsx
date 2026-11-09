"use client";

import Link from "next/link";
import { usePathname } from "next/navigation";
import type { ReactNode } from "react";
import clsx from "clsx";
import { Boxes, LayoutDashboard, LogOut, Package, ReceiptText, Store, Tag, Tags } from "lucide-react";
import { RequireAdmin } from "@/components/auth/guards";
import { useAuth } from "@/providers/auth-provider";

const NAV = [
  { href: "/admin", label: "Resumen", icon: LayoutDashboard, exact: true },
  { href: "/admin/products", label: "Productos", icon: Package },
  { href: "/admin/inventory", label: "Inventario", icon: Boxes },
  { href: "/admin/orders", label: "Pedidos", icon: ReceiptText },
  { href: "/admin/coupons", label: "Cupones", icon: Tag },
  { href: "/admin/categories", label: "Categorias", icon: Tags },
];

function Shell({ children }: { children: ReactNode }) {
  const pathname = usePathname();
  const { user, logout } = useAuth();

  return (
    <div className="min-h-screen lg:grid lg:grid-cols-[240px_1fr]">
      <aside className="border-b border-slate-200 bg-white lg:min-h-screen lg:border-b-0 lg:border-r">
        <div className="flex items-center justify-between px-4 py-4 lg:block">
          <Link href="/admin" className="flex items-center gap-2 text-lg font-bold">
            <Store className="h-6 w-6 text-brand-600" aria-hidden /> Administracion
          </Link>
          <p className="hidden pt-1 text-xs text-slate-500 lg:block">{user?.email}</p>
        </div>
        <nav aria-label="Administracion" className="flex gap-1 overflow-x-auto px-3 pb-3 lg:flex-col lg:overflow-visible lg:pb-0">
          {NAV.map(({ href, label, icon: Icon, exact }) => {
            const active = exact ? pathname === href : pathname.startsWith(href);
            return (
              <Link
                key={href}
                href={href}
                aria-current={active ? "page" : undefined}
                className={clsx(
                  "flex shrink-0 items-center gap-2 rounded-lg px-3 py-2 text-sm font-medium",
                  active ? "bg-brand-50 text-brand-700" : "text-slate-600 hover:bg-slate-100",
                )}
              >
                <Icon className="h-4 w-4" aria-hidden /> {label}
              </Link>
            );
          })}
          <Link href="/" className="flex shrink-0 items-center gap-2 rounded-lg px-3 py-2 text-sm font-medium text-slate-600 hover:bg-slate-100 lg:mt-6">
            <Store className="h-4 w-4" aria-hidden /> Ver tienda
          </Link>
          <button onClick={logout} className="flex shrink-0 items-center gap-2 rounded-lg px-3 py-2 text-left text-sm font-medium text-slate-600 hover:bg-slate-100">
            <LogOut className="h-4 w-4" aria-hidden /> Cerrar sesion
          </button>
        </nav>
      </aside>
      <main className="min-w-0 px-4 py-8 sm:px-8">{children}</main>
    </div>
  );
}

export default function AdminLayout({ children }: { children: ReactNode }) {
  return (
    <RequireAdmin>
      <Shell>{children}</Shell>
    </RequireAdmin>
  );
}
