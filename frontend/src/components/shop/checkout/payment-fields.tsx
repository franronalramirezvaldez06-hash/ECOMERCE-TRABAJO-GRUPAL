import { CreditCard } from "lucide-react";
import { SelectField, TextField } from "@/components/ui/form";
import { Card } from "@/components/ui/card";

const TEST_TOKENS = [
  { value: "tok_ok", label: "tok_ok - pago aprobado" },
  { value: "tok_declined", label: "tok_declined - tarjeta rechazada" },
  { value: "tok_flaky", label: "tok_flaky - falla 2 veces y se reintenta" },
  { value: "tok_slow", label: "tok_slow - pasarela lenta (timeout)" },
];

interface PaymentFieldsProps {
  gateways: string[];
  gateway: string;
  token: string;
  onGatewayChange: (gateway: string) => void;
  onTokenChange: (token: string) => void;
}

export function PaymentFields({ gateways, gateway, token, onGatewayChange, onTokenChange }: PaymentFieldsProps) {
  const isTestToken = TEST_TOKENS.some((t) => t.value === token);
  return (
    <Card className="space-y-4 p-5">
      <h2 className="flex items-center gap-2 font-semibold">
        <CreditCard className="h-5 w-5 text-brand-600" aria-hidden /> Pago
      </h2>
      <div className="grid gap-4 sm:grid-cols-2">
        <SelectField label="Pasarela de pago" value={gateway} onChange={(e) => onGatewayChange(e.target.value)} required>
          {gateways.map((g) => (
            <option key={g} value={g}>
              {g}
            </option>
          ))}
        </SelectField>
        <SelectField label="Medio de pago de prueba" value={isTestToken ? token : ""} onChange={(e) => e.target.value && onTokenChange(e.target.value)}>
          {TEST_TOKENS.map((t) => (
            <option key={t.value} value={t.value}>
              {t.label}
            </option>
          ))}
          <option value="">Otro (escribir abajo)</option>
        </SelectField>
        <div className="sm:col-span-2">
          <TextField
            label="Token de pago"
            value={token}
            onChange={(e) => onTokenChange(e.target.value)}
            required
            hint="Con Stripe real usa un PaymentMethod (ej. pm_card_visa). Con las pasarelas simuladas usa los tokens de arriba."
          />
        </div>
      </div>
    </Card>
  );
}
