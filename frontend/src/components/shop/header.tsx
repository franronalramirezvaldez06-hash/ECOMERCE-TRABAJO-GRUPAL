"use client";

import Link from "next/link";
import { usePathname, useRouter } from "next/navigation";
import { useState } from "react";
import clsx from "clsx";
import { LogOut, Menu, ShieldCheck, ShoppingCart, Store, User as UserIcon, X } from "lucide-react";
import { useAuth } from "@/providers/auth-provider";
import { useCart } from "@/providers/cart-provider";
import { LinkButton } from "@/components/ui/button";

interface NavItem {
  href: string;
  label: string;
}

const MOBILE_LINK = "block rounded-lg px-3 py-2 text-sm font-medium text-slate-700 hover:bg-slate-100";

function DesktopNav({ items }: { items: NavItem[] }) {
  const pathname = usePathname();
  return (
    <nav className="hidden items-center gap-1 md:flex" aria-label="Principal">
      {items.map((item) => (
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
  );
}

function CartLink() {
  const { count, ready } = useCart();
  return (
    <Link href="/cart" className="relative rounded-lg p-2 text-slate-700 hover:bg-slate-100" aria-label={`Carrito, ${ready ? count : 0} articulos`}>
      <ShoppingCart className="h-5 w-5" />
      {ready && count > 0 && (
        <span className="absolute -right-0.5 -top-0.5 flex h-5 min-w-5 items-center justify-center rounded-full bg-brand-600 px-1 text-xs font-semibold text-white">
          {count}
        </span>
      )}
    </Link>
  );
}

function AccountActions({ onSignOut }: { onSignOut: () => void }) {
  const { user, loading, isAdmin } = useAuth();
  if (loading) return null;
  if (!user) {
    return (
      <>
        <LinkButton href="/login" variant="ghost" size="sm">
          Ingresar
        </LinkButton>
        <LinkButton href="/register" size="sm">
          Crear cuenta
        </LinkButton>
      </>
    );
  }
  return (
    <>
      {isAdmin && (
        <LinkButton href="/admin" variant="secondary" size="sm">
          <ShieldCheck className="h-4 w-4" aria-hidden /> Admin
        </LinkButton>
      )}
      <span className="flex items-center gap-1.5 px-2 text-sm text-slate-600">
        <UserIcon className="h-4 w-4" aria-hidden /> {user.name}
      </span>
      <button onClick={onSignOut} className="rounded-lg p-2 text-slate-500 hover:bg-slate-100" aria-label="Cerrar sesion">
        <LogOut className="h-4 w-4" />
      </button>
    </>
  );
}

interface MobileMenuProps {
  items: NavItem[];
  onNavigate: () => void;
  onSignOut: () => void;
}

function MobileMenu({ items, onNavigate, onSignOut }: MobileMenuProps) {
  const { user, isAdmin } = useAuth();
  const links: NavItem[] = [
    ...items,
    ...(user && isAdmin ? [{ href: "/admin", label: "Panel de administracion" }] : []),
    ...(user ? [] : [{ href: "/login", label: "Ingresar" }]),
  ];
  return (
    <div className="space-y-1 border-t border-slate-200 bg-white px-4 py-3 md:hidden">
      {links.map((item) => (
        <Link key={item.href} href={item.href} onClick={onNavigate} className={MOBILE_LINK}>
          {item.label}
        </Link>
      ))}
      {user ? (
        <button onClick={onSignOut} className={clsx(MOBILE_LINK, "w-full text-left")}>
          Cerrar sesion ({user.name})
        </button>
      ) : (
        <Link href="/register" onClick={onNavigate} className="block rounded-lg px-3 py-2 text-sm font-medium text-brand-700 hover:bg-brand-50">
          Crear cuenta
        </Link>
      )}
    </div>
  );
}

export function Header() {
  const { user, logout } = useAuth();
  const router = useRouter();
  const [open, setOpen] = useState(false);

  const nav: NavItem[] = [
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
        <DesktopNav items={nav} />
        <div className="flex items-center gap-2">
          <CartLink />
          <div className="hidden items-center gap-2 md:flex">
            <AccountActions onSignOut={signOut} />
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
      {open && <MobileMenu items={nav} onNavigate={() => setOpen(false)} onSignOut={signOut} />}
    </header>
  );
}
