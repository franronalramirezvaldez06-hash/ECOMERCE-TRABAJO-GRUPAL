// Tipos que reflejan las respuestas JSON del backend en Go. Montos en centavos.

export type Role = "admin" | "customer";

export interface User {
  id: string;
  email: string;
  name: string;
  role: Role;
  active: boolean;
  created_at: string;
}

export interface LoginResult {
  token: string;
  expires_at: string;
  user: User;
}

export interface Product {
  id: string;
  sku: string;
  name: string;
  description: string;
  price_cents: number;
  kind: "simple" | "bundle";
  bundle_discount_pct: number;
  active: boolean;
  category_id: string | null;
  image_url: string;
  created_at: string;
}

/** Producto del listado con precio final y disponibilidad ya calculados. */
export interface ProductListItem extends Product {
  price_cents_total: number;
  available: number;
}

export interface BundleNode {
  id: string;
  sku: string;
  name: string;
  kind: "simple" | "bundle";
  quantity: number;
  price_cents: number;
  children?: BundleNode[];
}

export interface ProductDetail extends ProductListItem {
  tree: BundleNode;
}

export interface Category {
  id: string;
  name: string;
  created_at: string;
}

export interface QuoteLine {
  concept: string;
  amount_cents: number;
}

export interface Quote {
  currency: string;
  subtotal_cents: number;
  discount_cents: number;
  shipping_cents: number;
  tax_cents: number;
  total_cents: number;
  lines: QuoteLine[];
}

export type OrderStatus =
  | "pending"
  | "paid"
  | "failed"
  | "shipped"
  | "delivered"
  | "refunded"
  | "cancelled";

export interface Shipping {
  name: string;
  address: string;
  city: string;
  country: string;
  phone: string;
}

export interface OrderItem {
  product_id: string;
  name: string;
  unit_price_cents: number;
  quantity: number;
}

export interface Payment {
  id: string;
  gateway: string;
  status: "approved" | "declined" | "error" | "refunded";
  amount_cents: number;
  external_ref?: string;
  error_message?: string;
  created_at: string;
}

export interface Order {
  id: string;
  user_id?: string;
  customer_email: string;
  status: OrderStatus;
  coupon?: string;
  subtotal_cents: number;
  discount_cents: number;
  shipping_cents: number;
  tax_cents: number;
  total_cents: number;
  currency: string;
  shipping: Shipping;
  items?: OrderItem[];
  payment?: Payment;
  created_at: string;
}

export interface CheckoutPayload {
  items: { product_id: string; quantity: number }[];
  coupon?: string;
  gateway: string;
  payment_token: string;
  shipping: Shipping;
}

export interface Coupon {
  code: string;
  discount_pct: number;
  max_uses: number;
  used_count: number;
  expires_at: string | null;
  active: boolean;
  created_at: string;
}

export interface LowStockItem {
  product_id: string;
  sku: string;
  name: string;
  quantity: number;
}

export interface Movement {
  id: number;
  delta: number;
  reason: string;
  reference?: string;
  created_at: string;
}

export interface SalesSummary {
  from: string;
  to: string;
  orders_by_status: Record<string, number>;
  revenue_cents: number;
  refunded_cents: number;
  daily: { day: string; orders: number; revenue_cents: number }[];
  top_products: { product_id: string; name: string; units: number; revenue_cents: number }[];
}
