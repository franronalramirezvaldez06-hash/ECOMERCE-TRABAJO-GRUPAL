import type { OrderStatus } from "./api/types";

/** Formatea centavos como moneda (el backend siempre trabaja en centavos). */
export function formatMoney(cents: number, currency = "USD"): string {
  return new Intl.NumberFormat("es-CO", {
    style: "currency",
    currency,
    minimumFractionDigits: 2,
  }).format(cents / 100);
}

/** Convierte lo que escribe el usuario ("12,50" o "12.5") a centavos enteros. */
export function parseMoneyToCents(input: string): number | null {
  const normalized = input.trim().replace(",", ".");
  if (!/^\d+(\.\d{1,2})?$/.test(normalized)) return null;
  return Math.round(parseFloat(normalized) * 100);
}

export function centsToInput(cents: number): string {
  return (cents / 100).toFixed(2);
}

export function formatDate(iso: string): string {
  return new Intl.DateTimeFormat("es-CO", { dateStyle: "medium", timeStyle: "short" }).format(new Date(iso));
}

export const STATUS_LABELS: Record<OrderStatus, string> = {
  pending: "Pendiente",
  paid: "Pagada",
  failed: "Pago fallido",
  shipped: "Enviada",
  delivered: "Entregada",
  refunded: "Reembolsada",
  cancelled: "Cancelada",
};

export const STATUS_TONES: Record<OrderStatus, "gray" | "green" | "red" | "blue" | "amber" | "purple"> = {
  pending: "amber",
  paid: "green",
  failed: "red",
  shipped: "blue",
  delivered: "purple",
  refunded: "gray",
  cancelled: "gray",
};

/** Genera un UUID para la cabecera Idempotency-Key. */
export function newIdempotencyKey(): string {
  if (typeof crypto !== "undefined" && "randomUUID" in crypto) return crypto.randomUUID();
  return `${Date.now()}-${Math.random().toString(36).slice(2)}`;
}
