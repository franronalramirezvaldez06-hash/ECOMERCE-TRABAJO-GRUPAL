"use client";

import Link from "next/link";
import { usePathname, useRouter } from "next/navigation";
import { useState } from "react";
import clsx from "clsx";
import { LogOut, Menu, ShieldCheck, ShoppingCart, Store, User as UserIcon, X } from "lucide-react";
import { useAuth } from "@/providers/auth-provider";
import { useCart } from "@/providers/cart-provider";
import { LinkButton } from "@/components/ui/button";

export function Header() {
  const { user, loading, isAdmin, logout } = useAuth();
  const { count, ready } = useCart();
  const pathname = usePathname();
  const router = useRouter();
  const [open, setOpen] = useState(false);

  const nav = [
    { href: "/products", label: "Productos" },
    ...(user ? [{ href: "/orders", label: "Mis pedidos" }] : []),
  ];

  const signOut = () => {
    logout();
    setOpen(false);
    router.push("/");
  };

  return (
    <header className="sticky top-0 z-40 border-b border-slate-200 bg-white/90 backdrop-blur">
      <div className="mx-auto flex h-16 max-w-7xl items-center justify-between px-4 sm:px-6">
        <Link href="/" className="flex items-center gap-2 text-lg font-bold text-slate-900">
          <Store className="h-6 w-6 text-brand-600" aria-hidden />
          Tienda
        </Link>

        <nav className="hidden items-center gap-1 md:flex" aria-label="Principal">
          {nav.map((item) => (
            <Link
              key={item.href}
              href={item.href}
              className={clsx(
                "rounded-lg px-3 py-2 text-sm font-medium",
                pathname.startsWith(item.href) ? "bg-brand-50 text-brand-700" : "text-slate-600 hover:bg-slate-100",
              )}
            >
              {item.label}
            </Link>
          ))}
        </nav>

        <div className="flex items-center gap-2">
          <Link href="/cart" className="relative rounded-lg p-2 text-slate-700 hover:bg-slate-100" aria-label={`Carrito, ${ready ? count : 0} articulos`}>
            <ShoppingCart className="h-5 w-5" />
            {ready && count > 0 && (
              <span className="absolute -right-0.5 -top-0.5 flex h-5 min-w-5 items-center justify-center rounded-full bg-brand-600 px-1 text-xs font-semibold text-white">
                {count}
              </span>
            )}
          </Link>

          <div className="hidden items-center gap-2 md:flex">
            {loading ? null : user ? (
              <>
                {isAdmin && (
                  <LinkButton href="/admin" variant="secondary" size="sm">
                    <ShieldCheck className="h-4 w-4" aria-hidden /> Admin
                  </LinkButton>
                )}
                <span className="flex items-center gap-1.5 px-2 text-sm text-slate-600">
                  <UserIcon className="h-4 w-4" aria-hidden /> {user.name}
                </span>
                <button onClick={signOut} className="rounded-lg p-2 text-slate-500 hover:bg-slate-100" aria-label="Cerrar sesion">
                  <LogOut className="h-4 w-4" />
                </button>
              </>
            ) : (
              <>
                <LinkButton href="/login" variant="ghost" size="sm">
                  Ingresar
                </LinkButton>
                <LinkButton href="/register" size="sm">
                  Crear cuenta
                </LinkButton>
              </>
            )}
          </div>

          <button
            className="rounded-lg p-2 text-slate-700 hover:bg-slate-100 md:hidden"
            onClick={() => setOpen((v) => !v)}
            aria-label={open ? "Cerrar menu" : "Abrir menu"}
            aria-expanded={open}
          >
            {open ? <X className="h-5 w-5" /> : <Menu className="h-5 w-5" />}
          </button>
        </div>
      </div>

      {open && (
        <div className="space-y-1 border-t border-slate-200 bg-white px-4 py-3 md:hidden">
          {nav.map((item) => (
            <Link key={item.href} href={item.href} onClick={() => setOpen(false)} className="block rounded-lg px-3 py-2 text-sm font-medium text-slate-700 hover:bg-slate-100">
              {item.label}
            </Link>
          ))}
          {user ? (
            <>
              {isAdmin && (
                <Link href="/admin" onClick={() => setOpen(false)} className="block rounded-lg px-3 py-2 text-sm font-medium text-slate-700 hover:bg-slate-100">
                  Panel de administracion
                </Link>
              )}
              <button onClick={signOut} className="block w-full rounded-lg px-3 py-2 text-left text-sm font-medium text-slate-700 hover:bg-slate-100">
                Cerrar sesion ({user.name})
              </button>
            </>
          ) : (
            <>
              <Link href="/login" onClick={() => setOpen(false)} className="block rounded-lg px-3 py-2 text-sm font-medium text-slate-700 hover:bg-slate-100">
                Ingresar
              </Link>
              <Link href="/register" onClick={() => setOpen(false)} className="block rounded-lg px-3 py-2 text-sm font-medium text-brand-700 hover:bg-brand-50">
                Crear cuenta
              </Link>
            </>
          )}
        </div>
      )}
    </header>
  );
}
