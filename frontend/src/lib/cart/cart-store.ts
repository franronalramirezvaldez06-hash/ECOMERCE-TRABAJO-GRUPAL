// Logica pura del carrito (sin React) para poder probarla facilmente.

export interface CartLine {
  productId: string;
  name: string;
  /** Precio unitario en centavos (solo referencia visual: el precio real lo calcula el backend). */
  unitPriceCents: number;
  imageUrl: string;
  quantity: number;
  /** Unidades disponibles cuando se agrego, para limitar el stepper. */
  maxQuantity: number;
}

export type CartState = { lines: CartLine[]; coupon: string };

export const EMPTY_CART: CartState = { lines: [], coupon: "" };

export type CartAction =
  | { type: "add"; line: Omit<CartLine, "quantity">; quantity: number }
  | { type: "setQuantity"; productId: string; quantity: number }
  | { type: "remove"; productId: string }
  | { type: "setCoupon"; coupon: string }
  | { type: "clear" }
  | { type: "hydrate"; state: CartState };

const clamp = (value: number, max: number) => Math.max(1, Math.min(Math.floor(value), Math.max(1, max)));

export function cartReducer(state: CartState, action: CartAction): CartState {
  switch (action.type) {
    case "add": {
      const existing = state.lines.find((l) => l.productId === action.line.productId);
      if (existing) {
        return {
          ...state,
          lines: state.lines.map((l) =>
            l.productId === existing.productId
              ? {
                  ...l,
                  ...action.line,
                  quantity: clamp(l.quantity + action.quantity, action.line.maxQuantity),
                }
              : l,
          ),
        };
      }
      return {
        ...state,
        lines: [...state.lines, { ...action.line, quantity: clamp(action.quantity, action.line.maxQuantity) }],
      };
    }
    case "setQuantity":
      return {
        ...state,
        lines: state.lines.map((l) =>
          l.productId === action.productId ? { ...l, quantity: clamp(action.quantity, l.maxQuantity) } : l,
        ),
      };
    case "remove":
      return { ...state, lines: state.lines.filter((l) => l.productId !== action.productId) };
    case "setCoupon":
      return { ...state, coupon: action.coupon.trim().toUpperCase() };
    case "clear":
      return EMPTY_CART;
    case "hydrate":
      return action.state;
  }
}

export const cartCount = (state: CartState) => state.lines.reduce((sum, l) => sum + l.quantity, 0);

export const cartSubtotalCents = (state: CartState) =>
  state.lines.reduce((sum, l) => sum + l.unitPriceCents * l.quantity, 0);

/** Valida lo leido de localStorage: datos corruptos no deben romper la app. */
export function parseStoredCart(raw: string | null): CartState {
  if (!raw) return EMPTY_CART;
  try {
    const data = JSON.parse(raw) as Partial<CartState>;
    const lines = Array.isArray(data.lines)
      ? data.lines.filter(
          (l): l is CartLine =>
            !!l &&
            typeof l.productId === "string" &&
            typeof l.name === "string" &&
            typeof l.unitPriceCents === "number" &&
            typeof l.quantity === "number" &&
            l.quantity > 0 &&
            typeof l.maxQuantity === "number",
        )
      : [];
    return { lines, coupon: typeof data.coupon === "string" ? data.coupon : "" };
  } catch {
    return EMPTY_CART;
  }
}
