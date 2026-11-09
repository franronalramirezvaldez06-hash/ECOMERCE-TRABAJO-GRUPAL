"use client";

import { Button } from "@/components/ui/button";

export default function GlobalError({ reset }: { error: Error; reset: () => void }) {
  return (
    <main className="flex min-h-screen flex-col items-center justify-center px-4 text-center">
      <h1 className="text-2xl font-bold">Algo salio mal</h1>
      <p className="mt-1 text-slate-500">Ocurrio un error inesperado. Intenta de nuevo.</p>
      <Button className="mt-6" onClick={reset}>
        Reintentar
      </Button>
    </main>
  );
}
