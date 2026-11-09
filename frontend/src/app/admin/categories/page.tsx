"use client";

import { useState, type FormEvent } from "react";
import { useMutation, useQuery, useQueryClient } from "@tanstack/react-query";
import { catalogApi, adminApi } from "@/lib/api/services";
import { errorMessage } from "@/lib/api/client";
import { formatDate } from "@/lib/format";
import { useToast } from "@/providers/toast-provider";
import { Button } from "@/components/ui/button";
import { TextField } from "@/components/ui/form";
import { Card, EmptyState, ErrorState, PageLoader } from "@/components/ui/misc";
import { PageHeader } from "@/components/admin/admin-ui";

export default function CategoriesPage() {
  const toast = useToast();
  const queryClient = useQueryClient();
  const [name, setName] = useState("");
  const query = useQuery({ queryKey: ["categories"], queryFn: catalogApi.categories });

  const create = useMutation({
    mutationFn: () => adminApi.createCategory(name),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ["categories"] });
      toast.success("Categoria creada");
      setName("");
    },
    onError: (e) => toast.error(errorMessage(e)),
  });

  const submit = (e: FormEvent) => {
    e.preventDefault();
    if (name.trim()) create.mutate();
  };

  return (
    <>
      <PageHeader title="Categorias" description="Organizan el catalogo y permiten filtrar en la tienda." />
      <form onSubmit={submit} className="mb-6 flex max-w-md items-end gap-2">
        <div className="flex-1">
          <TextField label="Nueva categoria" value={name} onChange={(e) => setName(e.target.value)} placeholder="Ej: Accesorios" />
        </div>
        <Button type="submit" loading={create.isPending} className="mb-px">
          Crear
        </Button>
      </form>

      {query.isLoading ? (
        <PageLoader />
      ) : query.isError ? (
        <ErrorState message={errorMessage(query.error)} onRetry={() => query.refetch()} />
      ) : !query.data || query.data.length === 0 ? (
        <EmptyState title="Aun no hay categorias" />
      ) : (
        <Card className="max-w-md divide-y divide-slate-100">
          {query.data.map((c) => (
            <div key={c.id} className="flex items-center justify-between p-4 text-sm">
              <span className="font-medium">{c.name}</span>
              <span className="text-xs text-slate-500">{formatDate(c.created_at)}</span>
            </div>
          ))}
        </Card>
      )}
    </>
  );
}
