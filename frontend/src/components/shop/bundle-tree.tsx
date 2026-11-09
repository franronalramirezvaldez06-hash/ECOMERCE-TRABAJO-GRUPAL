import { Package } from "lucide-react";
import type { BundleNode } from "@/lib/api/types";
import { formatMoney } from "@/lib/format";

/** Muestra el arbol Composite de un combo: componentes y sub-combos. */
export function BundleTree({ nodes, depth = 0 }: { nodes: BundleNode[]; depth?: number }) {
  return (
    <ul className={depth === 0 ? "space-y-2" : "mt-2 space-y-2 border-l border-slate-200 pl-4"}>
      {nodes.map((n) => (
        <li key={`${n.id}-${depth}`}>
          <div className="flex items-center justify-between gap-3 text-sm">
            <span className="flex items-center gap-2">
              {n.kind === "bundle" && <Package className="h-4 w-4 text-violet-600" aria-hidden />}
              <span className="font-medium text-slate-900">{n.name}</span>
              <span className="text-slate-500">x{n.quantity}</span>
            </span>
            <span className="text-slate-600">{formatMoney(n.price_cents * n.quantity)}</span>
          </div>
          {n.children && n.children.length > 0 && <BundleTree nodes={n.children} depth={depth + 1} />}
        </li>
      ))}
    </ul>
  );
}
