import { describe, expect, it } from "vitest";
import { byClient } from "@/features/erp/finance/components/project-profit";
import { cashMoved, previewDocument } from "@/features/erp/finance/pricing";
import type { ChargeType, TaxCode } from "@/features/erp/finance/types";

const tax = (over: Partial<TaxCode>): TaxCode => ({
  id: "t", code: "T", name: "T", kind: "VAT", scope: "BOTH", rate: "11", baseNumerator: 1, baseDenominator: 1, priceIncluded: false, accountSaleId: "", accountPurchaseId: "",
  reportCode: "", validFrom: "", validTo: "", isDefaultSale: false, isDefaultPurchase: false, isActive: true, notes: "", cashBasis: false, transitionAccountId: "", ...over,
});
const charge = (over: Partial<ChargeType>): ChargeType => ({
  id: "c", code: "C", name: "C", scope: "BOTH", computation: "FIXED", value: "0", taxable: false, isDeduction: false, accountSaleId: "", accountPurchaseId: "", isActive: true, notes: "", ...over,
});
const items = [{ description: "x", unit: "", qty: "1", unitPrice: "10000000" }];

// The same cases as internal/finance/domain/documents_test.go in the backend.
describe("previewDocument", () => {
  it("charges each bracket slice at its own rate", () => {
    const brackets = [{ upTo: "1000000", rate: "5" }, { upTo: "5000000", rate: "10" }, { upTo: "0", rate: "20" }];
    const p = previewDocument({ items, discountPercent: "0", taxIds: ["w"], charges: [] }, { taxes: [tax({ id: "w", kind: "WITHHOLDING", rate: "0", brackets })], charges: [] });
    expect(p.withholding).toBeCloseTo(50_000 + 400_000 + 1_000_000);
  });

  it("applies DPP nilai lain", () => {
    const p = previewDocument({ items, discountPercent: "0", taxIds: ["t"], charges: [] }, { taxes: [tax({ rate: "12", baseNumerator: 11, baseDenominator: 12 })], charges: [] });
    expect(p.tax).toBeCloseTo(1_100_000);
    expect(p.total).toBeCloseTo(11_100_000);
  });

  it("takes withholding from the base without VAT", () => {
    const taxes = [tax({ id: "v" }), tax({ id: "w", kind: "WITHHOLDING", rate: "2" })];
    const p = previewDocument({ items, discountPercent: "0", taxIds: ["v", "w"], charges: [] }, { taxes, charges: [] });
    expect(p.withholding).toBeCloseTo(200_000);
    expect(p.total - p.withholding).toBeCloseTo(10_900_000);
  });

  it("takes included VAT out of the price", () => {
    const p = previewDocument({ items: [{ ...items[0], unitPrice: "1110000" }], discountPercent: "0", taxIds: ["t"], charges: [] }, { taxes: [tax({ priceIncluded: true })], charges: [] });
    expect(p.subtotal).toBeCloseTo(1_000_000);
    expect(p.total).toBeCloseTo(1_110_000);
  });

  it("keeps charges on the right side of the tax base", () => {
    const charges = [charge({ id: "a", computation: "PERCENT", taxable: true }), charge({ id: "m", value: "10000" }), charge({ id: "d", taxable: true, isDeduction: true })];
    const p = previewDocument({ items, discountPercent: "0", taxIds: ["t"], charges: [{ chargeId: "a", value: "10" }, { chargeId: "m", value: "" }, { chargeId: "d", value: "500000" }] }, { taxes: [tax({})], charges });
    expect(p.tax).toBeCloseTo(1_155_000);
    expect(p.total).toBeCloseTo(10_500_000 + 1_155_000 + 10_000);
  });
});

describe("byClient", () => {
  it("merges projects per client and keeps untagged apart", () => {
    const row = (tagId: string, tagName: string, clientName: string, revenue: string, cost: string) => ({ tagId, tagName, clientName, revenue, cost, profit: String(Number(revenue) - Number(cost)) });
    const merged = byClient([row("a", "SEO", "PT A", "100", "40"), row("b", "Ads", "PT A", "50", "10"), row("c", "Lepas", "", "30", "5"), row("", "", "", "0", "20")]);
    expect(merged.find((r) => r.tagName === "PT A")).toMatchObject({ revenue: "150.00", cost: "50.00", profit: "100.00" });
    expect(merged.find((r) => r.tagName === "Lepas")?.profit).toBe("25.00");
    expect(merged.find((r) => r.tagName === "")?.cost).toBe("20.00");
  });
});

describe("cashMoved", () => {
  const fee = { kind: "FEE", amount: "6500" };
  const pph = { kind: "WITHHOLDING", amount: "200000" };
  it("lowers a receipt by the fee and the withheld tax", () => {
    expect(cashMoved("AR", 10_000_000, [fee, pph])).toBe(9_793_500);
  });
  it("raises a vendor payment by the fee and lowers it by the withheld tax", () => {
    expect(cashMoved("AP", 10_000_000, [fee, pph])).toBe(9_806_500);
  });
});
