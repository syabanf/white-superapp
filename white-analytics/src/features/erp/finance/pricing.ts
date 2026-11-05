import type { DocumentValues } from "./actions";
import type { ChargeType, LedgerKind, TaxCode } from "./types";
import { parseMoney } from "@/features/erp/shared/money";

const num = (v: string | number | undefined) => Number(v) || 0;

/** Tax of one code on a base: its DPP share, then the flat rate or the brackets. */
export function taxOn(t: TaxCode, base: number) {
  const share = t.baseDenominator > 0 ? (base * t.baseNumerator) / t.baseDenominator : base;
  if (!t.brackets?.length) return (share * num(t.rate)) / 100;
  let tax = 0;
  let floor = 0;
  for (const b of t.brackets) {
    const top = num(b.upTo) === 0 || num(b.upTo) > share ? share : num(b.upTo);
    if (top > floor) {
      tax += ((top - floor) * num(b.rate)) / 100;
      floor = top;
    }
  }
  return tax;
}

/**
 * What the form shows while typing, in the same order as the backend's
 * Document.Recalculate: lines, included VAT out, discount, taxable charges, tax
 * base, taxes, untaxed charges. The backend's result is the one stored.
 */
export function previewDocument(v: Pick<DocumentValues, "items" | "discountPercent" | "taxIds" | "charges">, master: { taxes: TaxCode[]; charges: ChargeType[] }) {
  const gross = v.items.reduce((sum, it) => sum + num(it.qty) * num(it.unitPrice), 0);
  const taxes = v.taxIds.flatMap((id) => master.taxes.filter((t) => t.id === id));
  const ratio = (t: TaxCode) => (t.baseDenominator > 0 ? t.baseNumerator / t.baseDenominator : 1);
  const included = taxes.find((t) => t.priceIncluded && t.kind === "VAT");
  const subtotal = included ? gross / (1 + (ratio(included) * num(included.rate)) / 100) : gross;
  const after = subtotal * (1 - num(v.discountPercent) / 100);

  let taxable = 0;
  let untaxed = 0;
  for (const c of v.charges) {
    const m = master.charges.find((x) => x.id === c.chargeId);
    if (!m) continue;
    const value = num(c.value) || num(m.value);
    const amount = (m.computation === "PERCENT" ? (after * value) / 100 : value) * (m.isDeduction ? -1 : 1);
    if (m.taxable) taxable += amount;
    else untaxed += amount;
  }
  const dpp = after + taxable;
  const of = (kind: TaxCode["kind"]) => taxes.filter((t) => t.kind === kind).reduce((sum, t) => sum + taxOn(t, dpp), 0);
  const tax = of("VAT");
  return { subtotal, tax, withholding: of("WITHHOLDING"), total: dpp + tax + untaxed };
}

/** Only a fee on a vendor payment adds to the cash; every other adjustment lowers it. */
export function cashMoved(kind: LedgerKind, settled: number, adjustments: { kind: string; amount: string }[]) {
  return adjustments.reduce((sum, a) => (a.kind === "FEE" && kind === "AP" ? sum + parseMoney(a.amount) : sum - parseMoney(a.amount)), settled);
}
