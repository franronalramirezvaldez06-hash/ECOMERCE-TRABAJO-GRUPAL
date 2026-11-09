import { LinkButton } from "@/components/ui/button";

export default function NotFound() {
  return (
    <main className="flex min-h-screen flex-col items-center justify-center px-4 text-center">
      <p className="text-sm font-semibold text-brand-600">404</p>
      <h1 className="mt-2 text-2xl font-bold">Pagina no encontrada</h1>
      <p className="mt-1 text-slate-500">La pagina que buscas no existe o fue movida.</p>
      <LinkButton href="/" className="mt-6">
        Ir al inicio
      </LinkButton>
    </main>
  );
}
