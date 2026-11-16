"use client";

import { useEffect, useState, type FormEvent } from "react";
import { useQuery } from "@tanstack/react-query";
import { salesApi } from "@/lib/api/services";
import type { Shipping } from "@/lib/api/types";
import { useAuth } from "@/providers/auth-provider";
import { useCart } from "@/providers/cart-provider";
import { RequireAuth } from "@/components/auth/guards";
import { LinkButton } from "@/components/ui/button";
import { PageLoader } from "@/components/ui/spinner";
import { EmptyState } from "@/components/ui/states";
import { OrderSummary } from "@/components/shop/checkout/order-summary";
import { PaymentFields } from "@/components/shop/checkout/payment-fields";
import { ShippingFields } from "@/components/shop/checkout/shipping-fields";
import { useCheckoutSubmit } from "@/components/shop/checkout/use-checkout-submit";

function CheckoutForm() {
  const { user } = useAuth();
  const { state, ready } = useCart();
  const gateways = useQuery({ queryKey: ["gateways"], queryFn: salesApi.gateways });
  const checkout = useCheckoutSubmit();

  const [shipping, setShipping] = useState<Shipping>({ name: user?.name ?? "", address: "", city: "", country: "", phone: "" });
  const [gateway, setGateway] = useState("");
  const [token, setToken] = useState("tok_ok");

  useEffect(() => {
    if (!gateway && gateways.data?.gateways[0]) setGateway(gateways.data.gateways[0]);
  }, [gateways.data, gateway]);

  if (!ready) return <PageLoader />;
  if (state.lines.length === 0) {
    return <EmptyState title="Tu carrito esta vacio" action={<LinkButton href="/products">Explorar productos</LinkButton>} />;
  }

  const submit = (e: FormEvent) => {
    e.preventDefault();
    void checkout.submit({ gateway, token, shipping });
  };

  return (
    <div className="space-y-6">
      <h1 className="text-2xl font-bold">Finalizar compra</h1>
      <form onSubmit={submit} className="grid gap-6 lg:grid-cols-[1fr_380px]">
        <div className="space-y-6">
          <ShippingFields value={shipping} onChange={setShipping} />
          <PaymentFields
            gateways={gateways.data?.gateways ?? []}
            gateway={gateway}
            token={token}
            onGatewayChange={setGateway}
            onTokenChange={setToken}
          />
        </div>
        <OrderSummary failure={checkout.failure} error={checkout.error} submitting={checkout.submitting} canPay={Boolean(gateway)} />
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
