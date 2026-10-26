import { describe, expect, it } from "vitest";
import { lineTotals, newLine, payloadLines, setLineAmount } from "./journal-form";

describe("journal form rules", () => {
  it("keeps one side per line", () => {
    const l = newLine({ credit: "500.00" });
    const [next] = setLineAmount([l], l.key, "debit", "100.00");
    expect(next.debit).toBe("100.00");
    expect(next.credit).toBe("");
  });

  it("totals in cents", () => {
    const t = lineTotals([newLine({ debit: "0.10" }), newLine({ debit: "0.20" }), newLine({ credit: "0.30" })]);
    expect(t).toEqual({ debit: 0.3, credit: 0.3, diff: 0 });
  });

  it("rejects unbalanced and duplicate lines", () => {
    expect(payloadLines([newLine({ accountId: "a", debit: "100" }), newLine({ accountId: "b", credit: "90" })])).toHaveProperty("error");
    expect(payloadLines([newLine({ accountId: "a", debit: "100" }), newLine({ accountId: "a", credit: "100" })])).toHaveProperty("error");
  });

  it("builds API lines and drops empty rows", () => {
    const res = payloadLines([
      newLine({ accountId: "a", debit: "100", memo: " x " }),
      newLine({ accountId: "b", credit: "100" }),
      newLine(),
    ]);
    expect(res).toEqual({
      lines: [
        { accountId: "a", entrySide: "DEBIT", amount: "100.00", memo: "x" },
        { accountId: "b", entrySide: "CREDIT", amount: "100.00", memo: "" },
      ],
    });
  });
});
