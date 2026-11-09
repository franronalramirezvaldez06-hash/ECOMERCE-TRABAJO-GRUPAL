"use client";

import { useEffect, type ReactNode } from "react";
import { usePathname, useRouter } from "next/navigation";
import { ShieldAlert } from "lucide-react";
import { useAuth } from "@/providers/auth-provider";
import { PageLoader } from "@/components/ui/misc";
import { LinkButton } from "@/components/ui/button";

/** Redirige a /login (recordando la ruta) si no hay sesion. */
export function RequireAuth({ children }: { children: ReactNode }) {
  const { user, loading } = useAuth();
  const router = useRouter();
  const pathname = usePathname();

  useEffect(() => {
    if (!loading && !user) router.replace(`/login?next=${encodeURIComponent(pathname)}`);
  }, [loading, user, router, pathname]);

  if (loading || !user) return <PageLoader />;
  return <>{children}</>;
}

/** Solo administradores. Los clientes ven un aviso (el backend igual rechaza con 403). */
export function RequireAdmin({ children }: { children: ReactNode }) {
  const { user, loading, isAdmin } = useAuth();
  const router = useRouter();
  const pathname = usePathname();

  useEffect(() => {
    if (!loading && !user) router.replace(`/login?next=${encodeURIComponent(pathname)}`);
  }, [loading, user, router, pathname]);

  if (loading || !user) return <PageLoader />;
  if (!isAdmin) {
    return (
      <div className="mx-auto flex max-w-md flex-col items-center px-4 py-24 text-center">
        <ShieldAlert className="h-10 w-10 text-amber-500" aria-hidden />
        <h1 className="mt-4 text-xl font-semibold">Acceso restringido</h1>
        <p className="mt-1 text-sm text-slate-500">Esta seccion es solo para administradores.</p>
        <LinkButton href="/" className="mt-6">
          Volver a la tienda
        </LinkButton>
      </div>
    );
  }
  return <>{children}</>;
}
