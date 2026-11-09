"use client";

import { useEffect, useRef, useState, type FormEvent } from "react";
import { useRouter } from "next/navigation";
import { useQuery, useQueryClient } from "@tanstack/react-query";
import { CreditCard, Lock } from "lucide-react";
import { salesApi } from "@/lib/api/services";
import { ApiError, errorMessage } from "@/lib/api/client";
import type { Order, Shipping } from "@/lib/api/types";
import { newIdempotencyKey } from "@/lib/format";
import { useAuth } from "@/providers/auth-provider";
import { useCart } from "@/providers/cart-provider";
import { useToast } from "@/providers/toast-provider";
import { RequireAuth } from "@/components/auth/guards";
import { Button, LinkButton } from "@/components/ui/button";
import { SelectField, TextField } from "@/components/ui/form";
import { Card, EmptyState, PageLoader } from "@/components/ui/misc";
import { QuoteSummary, useQuote } from "@/components/shop/quote-summary";

const TEST_TOKENS = [
  { value: "tok_ok", label: "tok_ok - pago aprobado" },
  { value: "tok_declined", label: "tok_declined - tarjeta rechazada" },
  { value: "tok_flaky", label: "tok_flaky - falla 2 veces y se reintenta" },
  { value: "tok_slow", label: "tok_slow - pasarela lenta (timeout)" },
];

function CheckoutForm() {
  const router = useRouter();
  const toast = useToast();
  const queryClient = useQueryClient();
  const { user } = useAuth();
  const { state, dispatch, ready } = useCart();
  const quote = useQuote();
  const gateways = useQuery({ queryKey: ["gateways"], queryFn: salesApi.gateways });

  const [shipping, setShipping] = useState<Shipping>({ name: user?.name ?? "", address: "", city: "", country: "", phone: "" });
  const [gateway, setGateway] = useState("");
  const [token, setToken] = useState("tok_ok");
  const [submitting, setSubmitting] = useState(false);
  const [failure, setFailure] = useState<string | null>(null);
  const [error, setError] = useState<string | null>(null);

  // Misma clave en reintentos por error de red; nueva tras cada respuesta definitiva.
  const idempotencyKey = useRef(newIdempotencyKey());

  useEffect(() => {
    if (!gateway && gateways.data?.gateways[0]) setGateway(gateways.data.gateways[0]);
  }, [gateways.data, gateway]);

  if (!ready) return <PageLoader />;
  if (state.lines.length === 0) {
    return (
      <EmptyState
        title="Tu carrito esta vacio"
        action={<LinkButton href="/products">Explorar productos</LinkButton>}
      />
    );
  }

  const set = (field: keyof Shipping) => (e: React.ChangeEvent<HTMLInputElement>) =>
    setShipping((s) => ({ ...s, [field]: e.target.value }));

  const submit = async (e: FormEvent) => {
    e.preventDefault();
    setError(null);
    setFailure(null);
    setSubmitting(true);
    try {
      const order = await salesApi.checkout(
        {
          items: state.lines.map((l) => ({ product_id: l.productId, quantity: l.quantity })),
          coupon: state.coupon || undefined,
          gateway,
          payment_token: token,
          shipping,
        },
        idempotencyKey.current,
      );
      idempotencyKey.current = newIdempotencyKey();
      dispatch({ type: "clear" });
      queryClient.invalidateQueries({ queryKey: ["products"] });
      queryClient.invalidateQueries({ queryKey: ["orders"] });
      toast.success("Compra realizada con exito");
      router.push(`/orders/${order.id}`);
    } catch (err) {
      if (err instanceof ApiError && err.status === 402) {
        // Pago rechazado: la orden quedo "failed" y el stock se devolvio.
        idempotencyKey.current = newIdempotencyKey();
        const order = err.body as Order | undefined;
        setFailure(order?.payment?.error_message ?? "El pago fue rechazado. Prueba con otro medio de pago.");
      } else if (err instanceof ApiError && err.status === 409) {
        idempotencyKey.current = newIdempotencyKey();
        setError(`${err.message}. Ajusta las cantidades de tu carrito.`);
      } else {
        // Error de red u otro: se conserva la clave para reintentar sin duplicar la compra.
        setError(errorMessage(err));
      }
    } finally {
      setSubmitting(false);
    }
  };

  return (
    <div className="space-y-6">
      <h1 className="text-2xl font-bold">Finalizar compra</h1>
      <form onSubmit={submit} className="grid gap-6 lg:grid-cols-[1fr_380px]">
        <div className="space-y-6">
          <Card className="space-y-4 p-5">
            <h2 className="font-semibold">Direccion de envio</h2>
            <div className="grid gap-4 sm:grid-cols-2">
              <TextField label="Nombre completo" value={shipping.name} onChange={set("name")} required autoComplete="name" />
              <TextField label="Telefono (opcional)" value={shipping.phone} onChange={set("phone")} autoComplete="tel" />
              <div className="sm:col-span-2">
                <TextField label="Direccion" value={shipping.address} onChange={set("address")} required autoComplete="street-address" />
              </div>
              <TextField label="Ciudad" value={shipping.city} onChange={set("city")} required autoComplete="address-level2" />
              <TextField label="Pais" value={shipping.country} onChange={set("country")} required autoComplete="country-name" />
            </div>
          </Card>

          <Card className="space-y-4 p-5">
            <h2 className="flex items-center gap-2 font-semibold">
              <CreditCard className="h-5 w-5 text-brand-600" aria-hidden /> Pago
            </h2>
            <div className="grid gap-4 sm:grid-cols-2">
              <SelectField label="Pasarela de pago" value={gateway} onChange={(e) => setGateway(e.target.value)} required>
                {gateways.data?.gateways.map((g) => (
                  <option key={g} value={g}>
                    {g}
                  </option>
                ))}
              </SelectField>
              <SelectField label="Medio de pago de prueba" value={TEST_TOKENS.some((t) => t.value === token) ? token : ""} onChange={(e) => e.target.value && setToken(e.target.value)}>
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
                  onChange={(e) => setToken(e.target.value)}
                  required
                  hint="Con Stripe real usa un PaymentMethod (ej. pm_card_visa). Con las pasarelas simuladas usa los tokens de arriba."
                />
              </div>
            </div>
          </Card>
        </div>

        <Card className="h-fit space-y-5 p-5">
          <h2 className="font-semibold">Resumen del pedido</h2>
          <ul className="space-y-1 text-sm text-slate-600">
            {state.lines.map((l) => (
              <li key={l.productId} className="flex justify-between gap-3">
                <span className="truncate">
                  {l.quantity} x {l.name}
                </span>
              </li>
            ))}
          </ul>
          <QuoteSummary quote={quote} />

          {failure && (
            <p role="alert" className="rounded-lg bg-red-50 p-3 text-sm text-red-700">
              <strong>Pago rechazado.</strong> {failure}
            </p>
          )}
          {error && (
            <p role="alert" className="rounded-lg bg-red-50 p-3 text-sm text-red-700">
              {error}
            </p>
          )}

          <Button type="submit" size="lg" className="w-full" loading={submitting} disabled={quote.isError || !gateway}>
            <Lock className="h-4 w-4" aria-hidden />
            Pagar ahora
          </Button>
          <p className="text-center text-xs text-slate-500">Si el pago falla, no se descuenta tu inventario ni tu cupon.</p>
        </Card>
      </form>
    </div>
  );
}

export default function CheckoutPage() {
  return (
    <RequireAuth>
      <CheckoutForm />
    </RequireAuth>
  );
}
