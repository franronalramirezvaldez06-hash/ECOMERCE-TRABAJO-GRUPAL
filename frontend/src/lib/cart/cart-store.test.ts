import { describe, expect, it } from "vitest";
import { cartCount, cartReducer, cartSubtotalCents, EMPTY_CART, parseStoredCart } from "./cart-store";

const mouse = { productId: "m", name: "Mouse", unitPriceCents: 2000, imageUrl: "", maxQuantity: 5 };

describe("cartReducer", () => {
  it("agrega y acumula cantidades de la misma linea", () => {
    let s = cartReducer(EMPTY_CART, { type: "add", line: mouse, quantity: 2 });
    s = cartReducer(s, { type: "add", line: mouse, quantity: 1 });
    expect(s.lines).toHaveLength(1);
    expect(cartCount(s)).toBe(3);
    expect(cartSubtotalCents(s)).toBe(6000);
  });

  it("no supera la disponibilidad", () => {
    let s = cartReducer(EMPTY_CART, { type: "add", line: mouse, quantity: 99 });
    expect(s.lines[0]?.quantity).toBe(5);
    s = cartReducer(s, { type: "setQuantity", productId: "m", quantity: 50 });
    expect(s.lines[0]?.quantity).toBe(5);
  });

  it("nunca baja de 1 por setQuantity y elimina con remove", () => {
    let s = cartReducer(EMPTY_CART, { type: "add", line: mouse, quantity: 2 });
    s = cartReducer(s, { type: "setQuantity", productId: "m", quantity: 0 });
    expect(s.lines[0]?.quantity).toBe(1);
    s = cartReducer(s, { type: "remove", productId: "m" });
    expect(s.lines).toHaveLength(0);
  });

  it("normaliza el cupon", () => {
    expect(cartReducer(EMPTY_CART, { type: "setCoupon", coupon: " verano20 " }).coupon).toBe("VERANO20");
  });
});

describe("parseStoredCart", () => {
  it("tolera datos corruptos", () => {
    expect(parseStoredCart(null)).toEqual(EMPTY_CART);
    expect(parseStoredCart("{no es json")).toEqual(EMPTY_CART);
    expect(parseStoredCart(JSON.stringify({ lines: [{ productId: 1 }] })).lines).toHaveLength(0);
  });

  it("recupera un carrito valido", () => {
    const stored = JSON.stringify({ lines: [{ ...mouse, quantity: 2 }], coupon: "X" });
    const s = parseStoredCart(stored);
    expect(s.lines[0]?.quantity).toBe(2);
    expect(s.coupon).toBe("X");
  });
});
