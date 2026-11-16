import { Pencil } from "lucide-react";
import type { ProductListItem } from "@/lib/api/types";
import { formatMoney } from "@/lib/format";
import { Badge } from "@/components/ui/badge";
import { Button } from "@/components/ui/button";
import { Table, td } from "@/components/admin/admin-ui";

const HEADERS = [
  { label: "Producto" },
  { label: "Tipo" },
  { label: "Precio", className: "text-right" },
  { label: "Disponible", className: "text-right" },
  { label: "Estado" },
  { label: "", className: "text-right" },
];

interface RowActions {
  onEdit: (product: ProductListItem) => void;
  onToggle: (product: ProductListItem) => void;
  toggling: boolean;
}

function ProductRow({ product: p, onEdit, onToggle, toggling }: RowActions & { product: ProductListItem }) {
  return (
    <tr className={p.active ? "" : "bg-slate-50 text-slate-500"}>
      <td className={td}>
        <p className="font-medium text-slate-900">{p.name}</p>
        <p className="text-xs text-slate-500">{p.sku}</p>
      </td>
      <td className={td}>{p.kind === "bundle" ? <Badge tone="purple">Combo</Badge> : <Badge>Simple</Badge>}</td>
      <td className={`${td} text-right`}>{formatMoney(p.price_cents_total)}</td>
      <td className={`${td} text-right ${p.available === 0 ? "font-semibold text-red-600" : ""}`}>{p.available}</td>
      <td className={td}>{p.active ? <Badge tone="green">Activo</Badge> : <Badge tone="amber">Inactivo</Badge>}</td>
      <td className={`${td} whitespace-nowrap text-right`}>
        <Button variant="ghost" size="sm" onClick={() => onEdit(p)} aria-label={`Editar ${p.name}`}>
          <Pencil className="h-4 w-4" aria-hidden /> Editar
        </Button>
        <Button variant="ghost" size="sm" onClick={() => onToggle(p)} disabled={toggling}>
          {p.active ? "Desactivar" : "Activar"}
        </Button>
      </td>
    </tr>
  );
}

export function ProductsTable({ items, ...actions }: RowActions & { items: ProductListItem[] }) {
  return (
    <Table headers={HEADERS}>
      {items.map((p) => (
        <ProductRow key={p.id} product={p} {...actions} />
      ))}
    </Table>
  );
}
