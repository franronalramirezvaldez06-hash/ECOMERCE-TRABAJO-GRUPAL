import { api } from "./client";
import type {
  Category,
  CheckoutPayload,
  Coupon,
  LoginResult,
  LowStockItem,
  Movement,
  Order,
  ProductDetail,
  ProductListItem,
  Product,
  Quote,
  SalesSummary,
  User,
} from "./types";

export interface ProductFilters {
  q?: string;
  category_id?: string;
  include_inactive?: boolean;
  limit?: number;
  offset?: number;
}

export const authApi = {
  login: (email: string, password: string) =>
    api<LoginResult>("/api/v1/auth/login", { method: "POST", body: { email, password } }),
  register: (input: { email: string; name: string; password: string }) =>
    api<User>("/api/v1/auth/register", { method: "POST", body: input }),
  me: () => api<User>("/api/v1/auth/me"),
};

export const catalogApi = {
  list: (filters: ProductFilters = {}) =>
    api<ProductListItem[]>("/api/v1/products", { query: { ...filters } }),
  get: (id: string) => api<ProductDetail>(`/api/v1/products/${id}`),
  categories: () => api<Category[]>("/api/v1/categories"),
};

export const salesApi = {
  quote: (items: { product_id: string; quantity: number }[], coupon?: string) =>
    api<Quote>("/api/v1/quotes", { method: "POST", body: { items, coupon: coupon ?? "" } }),
  gateways: () => api<{ gateways: string[] }>("/api/v1/payments/gateways"),
  checkout: (payload: CheckoutPayload, idempotencyKey: string) =>
    api<Order>("/api/v1/orders", {
      method: "POST",
      body: payload,
      headers: { "Idempotency-Key": idempotencyKey },
    }),
  orders: (params: { status?: string; limit?: number; offset?: number; user_id?: string } = {}) =>
    api<Order[]>("/api/v1/orders", { query: params }),
  order: (id: string) => api<Order>(`/api/v1/orders/${id}`),
  cancel: (id: string) => api<Order>(`/api/v1/orders/${id}/cancel`, { method: "POST" }),
};

export interface CreateProductInput {
  sku: string;
  name: string;
  description: string;
  price_cents: number;
  initial_stock: number;
  category_id: string;
  image_url: string;
}

export interface CreateBundleInput {
  sku: string;
  name: string;
  description: string;
  discount_pct: number;
  category_id: string;
  image_url: string;
  items: { product_id: string; quantity: number }[];
}

export interface UpdateProductInput {
  name?: string;
  description?: string;
  price_cents?: number;
  category_id?: string;
  image_url?: string;
  active?: boolean;
  bundle_discount_pct?: number;
}

export const adminApi = {
  createProduct: (input: CreateProductInput) =>
    api<Product>("/api/v1/products", { method: "POST", body: input }),
  createBundle: (input: CreateBundleInput) =>
    api<Product>("/api/v1/bundles", { method: "POST", body: input }),
  updateProduct: (id: string, input: UpdateProductInput) =>
    api<Product>(`/api/v1/products/${id}`, { method: "PUT", body: input }),
  createCategory: (name: string) => api<Category>("/api/v1/categories", { method: "POST", body: { name } }),

  stock: (id: string) => api<{ product_id: string; quantity: number }>(`/api/v1/inventory/${id}`),
  adjustStock: (id: string, delta: number, reason: string) =>
    api<{ product_id: string; quantity: number }>(`/api/v1/inventory/${id}/adjust`, {
      method: "POST",
      body: { delta, reason },
    }),
  setStock: (id: string, quantity: number) =>
    api<{ product_id: string; quantity: number }>(`/api/v1/inventory/${id}`, {
      method: "PUT",
      body: { quantity },
    }),
  movements: (id: string, limit = 20) => api<Movement[]>(`/api/v1/inventory/${id}/movements`, { query: { limit } }),
  lowStock: (threshold: number) => api<LowStockItem[]>("/api/v1/inventory/low-stock", { query: { threshold } }),

  coupons: () => api<Coupon[]>("/api/v1/coupons"),
  createCoupon: (input: { code: string; discount_pct: number; max_uses: number; expires_at: string | null }) =>
    api<Coupon>("/api/v1/coupons", { method: "POST", body: input }),
  setCouponActive: (code: string, active: boolean) =>
    api<{ code: string; active: boolean }>(`/api/v1/coupons/${encodeURIComponent(code)}`, {
      method: "PATCH",
      body: { active },
    }),

  refund: (id: string, reason: string) =>
    api<Order>(`/api/v1/orders/${id}/refund`, { method: "POST", body: { reason } }),
  ship: (id: string) => api<Order>(`/api/v1/orders/${id}/ship`, { method: "POST" }),
  deliver: (id: string) => api<Order>(`/api/v1/orders/${id}/deliver`, { method: "POST" }),

  salesReport: (from?: string, to?: string) => api<SalesSummary>("/api/v1/reports/sales", { query: { from, to } }),
};
