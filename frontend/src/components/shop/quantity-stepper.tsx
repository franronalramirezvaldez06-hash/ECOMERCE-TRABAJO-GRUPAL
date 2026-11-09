"use client";

import { Minus, Plus } from "lucide-react";

export function QuantityStepper({
  value,
  min = 1,
  max,
  onChange,
  label = "Cantidad",
}: {
  value: number;
  min?: number;
  max: number;
  onChange: (value: number) => void;
  label?: string;
}) {
  return (
    <div className="inline-flex items-center rounded-lg border border-slate-300 bg-white" role="group" aria-label={label}>
      <button
        type="button"
        onClick={() => onChange(value - 1)}
        disabled={value <= min}
        aria-label="Disminuir cantidad"
        className="flex h-9 w-9 items-center justify-center text-slate-600 hover:bg-slate-50 disabled:text-slate-300"
      >
        <Minus className="h-4 w-4" />
      </button>
      <span className="min-w-8 text-center text-sm font-medium tabular-nums" aria-live="polite">
        {value}
      </span>
      <button
        type="button"
        onClick={() => onChange(value + 1)}
        disabled={value >= max}
        aria-label="Aumentar cantidad"
        className="flex h-9 w-9 items-center justify-center text-slate-600 hover:bg-slate-50 disabled:text-slate-300"
      >
        <Plus className="h-4 w-4" />
      </button>
    </div>
  );
}
