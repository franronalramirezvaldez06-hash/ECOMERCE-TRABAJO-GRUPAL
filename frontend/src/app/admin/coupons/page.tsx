"use client";

import { useState, type FormEvent } from "react";
import { useMutation, useQuery, useQueryClient } from "@tanstack/react-query";
import { Plus } from "lucide-react";
import { adminApi } from "@/lib/api/services";
import { errorMessage } from "@/lib/api/client";
import { formatDate } from "@/lib/format";
import { useToast } from "@/providers/toast-provider";
import { Button } from "@/components/ui/button";
import { TextField } from "@/components/ui/form";
import { Badge, EmptyState, ErrorState, Modal, PageLoader } from "@/components/ui/misc";
import { PageHeader, Table, td } from "@/components/admin/admin-ui";

function CouponModal({ onClose }: { onClose: () => void }) {
  const toast = useToast();
  const queryClient = useQueryClient();
  const [form, setForm] = useState({ code: "", pct: "10", maxUses: "0", expires: "" });
  const [error, setError] = useState<string | null>(null);
  const set = (k: keyof typeof form) => (e: React.ChangeEvent<HTMLInputElement>) => setForm((f) => ({ ...f, [k]: e.target.value }));

  const create = useMutation({
    mutationFn: () =>
      adminApi.createCoupon({
        code: form.code,
        discount_pct: Number(form.pct),
        max_uses: Number(form.maxUses) || 0,
        // El cupon vence al final del dia elegido.
        expires_at: form.expires ? new Date(`${form.expires}T23:59:59`).toISOString() : null,
      }),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ["admin", "coupons"] });
      toast.success("Cupon creado");
      onClose();
    },
    onError: (e) => setError(errorMessage(e)),
  });

  const submit = (e: FormEvent) => {
    e.preventDefault();
    setError(null);
    create.mutate();
  };

  return (
    <Modal open title="Nuevo cupon" onClose={onClose}>
      <form onSubmit={submit} className="space-y-4">
        <TextField label="Codigo" value={form.code} onChange={set("code")} required className="uppercase" placeholder="VERANO25" />
        <div className="grid gap-4 sm:grid-cols-2">
          <TextField label="Descuento (%)" type="number" min={1} max={100} value={form.pct} onChange={set("pct")} required />
          <TextField label="Usos maximos" type="number" min={0} value={form.maxUses} onChange={set("maxUses")} hint="0 = ilimitado" />
        </div>
        <TextField label="Vence el (opcional)" type="date" value={form.expires} onChange={set("expires")} />
        {error && (
          <p role="alert" className="rounded-lg bg-red-50 p-3 text-sm text-red-700">
            {error}
          </p>
        )}
        <div className="flex justify-end gap-2">
          <Button variant="secondary" onClick={onClose}>
            Cancelar
          </Button>
          <Button type="submit" loading={create.isPending}>
            Crear
          </Button>
        </div>
      </form>
    </Modal>
  );
}

export default function CouponsPage() {
  const toast = useToast();
  const queryClient = useQueryClient();
  const [creating, setCreating] = useState(false);
  const query = useQuery({ queryKey: ["admin", "coupons"], queryFn: adminApi.coupons });

  const toggle = useMutation({
    mutationFn: ({ code, active }: { code: string; active: boolean }) => adminApi.setCouponActive(code, active),
    onSuccess: () => queryClient.invalidateQueries({ queryKey: ["admin", "coupons"] }),
    onError: (e) => toast.error(errorMessage(e)),
  });

  return (
    <>
      <PageHeader
        title="Cupones"
        description="Descuentos por porcentaje con vigencia y limite de usos."
        actions={
          <Button onClick={() => setCreating(true)}>
            <Plus className="h-4 w-4" aria-hidden /> Nuevo cupon
          </Button>
        }
      />
      {query.isLoading ? (
        <PageLoader />
      ) : query.isError ? (
        <ErrorState message={errorMessage(query.error)} onRetry={() => query.refetch()} />
      ) : !query.data || query.data.length === 0 ? (
        <EmptyState title="No hay cupones" />
      ) : (
        <Table headers={[{ label: "Codigo" }, { label: "Descuento" }, { label: "Usos" }, { label: "Vence" }, { label: "Estado" }, { label: "", className: "text-right" }]}>
          {query.data.map((c) => {
            const expired = c.expires_at !== null && new Date(c.expires_at) < new Date();
            const exhausted = c.max_uses > 0 && c.used_count >= c.max_uses;
            return (
              <tr key={c.code}>
                <td className={`${td} font-mono font-medium`}>{c.code}</td>
                <td className={td}>{c.discount_pct}%</td>
                <td className={td}>
                  {c.used_count}
                  {c.max_uses > 0 ? ` / ${c.max_uses}` : " / ∞"}
                </td>
                <td className={`${td} text-slate-500`}>{c.expires_at ? formatDate(c.expires_at) : "Sin vencimiento"}</td>
                <td className={td}>
                  {!c.active ? <Badge>Inactivo</Badge> : expired ? <Badge tone="amber">Vencido</Badge> : exhausted ? <Badge tone="amber">Agotado</Badge> : <Badge tone="green">Vigente</Badge>}
                </td>
                <td className={`${td} text-right`}>
                  <Button variant="ghost" size="sm" onClick={() => toggle.mutate({ code: c.code, active: !c.active })} disabled={toggle.isPending}>
                    {c.active ? "Desactivar" : "Activar"}
                  </Button>
                </td>
              </tr>
            );
          })}
        </Table>
      )}
      {creating && <CouponModal onClose={() => setCreating(false)} />}
    </>
  );
}
