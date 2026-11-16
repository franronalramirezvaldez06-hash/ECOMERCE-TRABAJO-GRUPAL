import type { Category } from "@/lib/api/types";
import { Button } from "@/components/ui/button";
import { SelectField } from "@/components/ui/form";

interface CategorySelectProps {
  value: string;
  onChange: (value: string) => void;
  categories: Category[] | undefined;
}

export function CategorySelect({ value, onChange, categories }: CategorySelectProps) {
  return (
    <SelectField label="Categoria" value={value} onChange={(e) => onChange(e.target.value)}>
      <option value="">Sin categoria</option>
      {categories?.map((c) => (
        <option key={c.id} value={c.id}>
          {c.name}
        </option>
      ))}
    </SelectField>
  );
}

export function FormError({ message }: { message: string | null }) {
  if (!message) return null;
  return (
    <p role="alert" className="rounded-lg bg-red-50 p-3 text-sm text-red-700">
      {message}
    </p>
  );
}

export function FormActions({ submitLabel, loading, onCancel }: { submitLabel: string; loading: boolean; onCancel: () => void }) {
  return (
    <div className="flex justify-end gap-2 pt-2">
      <Button variant="secondary" onClick={onCancel}>
        Cancelar
      </Button>
      <Button type="submit" loading={loading}>
        {submitLabel}
      </Button>
    </div>
  );
}
