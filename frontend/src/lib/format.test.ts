import { describe, expect, it } from "vitest";
import { centsToInput, parseMoneyToCents } from "./format";

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
