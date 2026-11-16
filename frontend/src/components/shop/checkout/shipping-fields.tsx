import type { ChangeEvent } from "react";
import type { Shipping } from "@/lib/api/types";
import { TextField } from "@/components/ui/form";
import { Card } from "@/components/ui/card";

export function ShippingFields({ value, onChange }: { value: Shipping; onChange: (next: Shipping) => void }) {
  const set = (field: keyof Shipping) => (e: ChangeEvent<HTMLInputElement>) =>
    onChange({ ...value, [field]: e.target.value });

  return (
    <Card className="space-y-4 p-5">
      <h2 className="font-semibold">Direccion de envio</h2>
      <div className="grid gap-4 sm:grid-cols-2">
        <TextField label="Nombre completo" value={value.name} onChange={set("name")} required autoComplete="name" />
        <TextField label="Telefono (opcional)" value={value.phone} onChange={set("phone")} autoComplete="tel" />
        <div className="sm:col-span-2">
          <TextField label="Direccion" value={value.address} onChange={set("address")} required autoComplete="street-address" />
        </div>
        <TextField label="Ciudad" value={value.city} onChange={set("city")} required autoComplete="address-level2" />
        <TextField label="Pais" value={value.country} onChange={set("country")} required autoComplete="country-name" />
      </div>
    </Card>
  );
}
