import type { DocumentValues } from "./actions";
import type { ChargeType, TaxCode } from "./types";

const num = (v: string | number | undefined) => Number(v) || 0;

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
  const of = (kind: TaxCode["kind"]) => taxes.filter((t) => t.kind === kind).reduce((sum, t) => sum + (dpp * ratio(t) * num(t.rate)) / 100, 0);
  const tax = of("VAT");
  return { subtotal, tax, withholding: of("WITHHOLDING"), total: dpp + tax + untaxed };
}
