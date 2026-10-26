import { describe, expect, it } from "vitest";
import { formatRupiah, parseMoney, parseRupiahInput, sumMoney, toMoneyString } from "./money";

describe("erp money helpers", () => {
  it("parses decimal strings", () => {
    expect(parseMoney("1250000.00")).toBe(1250000);
    expect(parseMoney("")).toBe(0);
    expect(parseMoney("abc")).toBe(0);
  });
  it("formats rupiah", () => {
    expect(formatRupiah("1250000.00")).toBe("Rp 1.250.000");
    expect(formatRupiah(-5000)).toBe("-Rp 5.000");
    expect(formatRupiah(5000, { symbol: false })).toBe("5.000");
  });
  it("round-trips user input", () => {
    expect(parseRupiahInput("Rp 1.250.000")).toBe(1250000);
    expect(parseRupiahInput("1.500,50")).toBe(1500.5);
    expect(toMoneyString(1500.5)).toBe("1500.50");
  });
  it("sums without float drift", () => {
    expect(sumMoney(["0.10", "0.20", 0.3])).toBe(0.6);
  });
});
