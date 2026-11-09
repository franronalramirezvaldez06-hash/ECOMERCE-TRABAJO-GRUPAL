"use client";

import { useState } from "react";
import clsx from "clsx";
import { ImageOff } from "lucide-react";

/** Imagen con respaldo si la URL falta o falla (las URLs las carga el admin). */
export function ProductImage({ src, alt, className }: { src: string; alt: string; className?: string }) {
  const [failed, setFailed] = useState(false);
  if (!src || failed) {
    return (
      <div className={clsx("flex items-center justify-center bg-slate-100 text-slate-300", className)} role="img" aria-label={alt}>
        <ImageOff className="h-8 w-8" aria-hidden />
      </div>
    );
  }
  return (
    // eslint-disable-next-line @next/next/no-img-element
    <img src={src} alt={alt} loading="lazy" onError={() => setFailed(true)} className={clsx("object-cover", className)} />
  );
}
