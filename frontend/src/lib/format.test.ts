import { describe, expect, it } from "vitest";
import { centsToInput, formatMoney, parseMoneyToCents } from "./format";

describe("formatMoney", () => {
  it("muestra soles por defecto y respeta la moneda de pedidos anteriores", () => {
    expect(formatMoney(1250)).toMatch(/^S\/\s?12\.50$/);
    expect(formatMoney(1250, "USD")).toContain("12.50");
    expect(formatMoney(1250, "USD")).not.toContain("S/");
  });
});

describe("parseMoneyToCents", () => {
  it("convierte formatos validos", () => {
    expect(parseMoneyToCents("12")).toBe(1200);
    expect(parseMoneyToCents("12.5")).toBe(1250);
    expect(parseMoneyToCents("12,50")).toBe(1250);
    expect(parseMoneyToCents(" 0.99 ")).toBe(99);
  });

  it("evita errores de punto flotante", () => {
    expect(parseMoneyToCents("19.99")).toBe(1999);
    expect(parseMoneyToCents("0.29")).toBe(29);
  });

  it("rechaza entradas invalidas", () => {
    expect(parseMoneyToCents("")).toBeNull();
    expect(parseMoneyToCents("abc")).toBeNull();
    expect(parseMoneyToCents("-5")).toBeNull();
    expect(parseMoneyToCents("1.234")).toBeNull();
  });
});

describe("centsToInput", () => {
  it("formatea con dos decimales", () => {
    expect(centsToInput(1250)).toBe("12.50");
    expect(centsToInput(5)).toBe("0.05");
  });
});
