import { formatRupiah, parseMoney, toMoneyString } from "@/features/erp/shared/money";
import type { JournalEntry, JournalLineInput } from "./types";
import { fs } from "./strings";

/** Pure state and rules of the manual journal form (unit tested). */

export type FormLine = { key: string; accountId: string; debit: string; credit: string; memo: string };

let seq = 0;
export function newLine(partial: Partial<FormLine> = {}): FormLine {
  seq += 1;
  return { key: `l${seq}`, accountId: "", debit: "", credit: "", memo: "", ...partial };
}

export function linesFromEntry(entry: JournalEntry): FormLine[] {
  return [...entry.lines]
    .sort((a, b) => a.sortOrder - b.sortOrder)
    .map((l) =>
      newLine({
        accountId: l.accountId,
        debit: l.entrySide === "DEBIT" ? l.amount : "",
        credit: l.entrySide === "CREDIT" ? l.amount : "",
        memo: l.memo,
      }),
    );
}

const cents = (v: string) => Math.round(parseMoney(v) * 100);

export function lineTotals(lines: FormLine[]) {
  let debit = 0;
  let credit = 0;
  for (const l of lines) {
    debit += Math.max(0, cents(l.debit));
    credit += Math.max(0, cents(l.credit));
  }
  return { debit: debit / 100, credit: credit / 100, diff: (debit - credit) / 100 };
}

/** One side per line: typing a debit clears the credit and vice versa. */
export function setLineAmount(lines: FormLine[], key: string, side: "debit" | "credit", value: string): FormLine[] {
  const other = side === "debit" ? "credit" : "debit";
  return lines.map((l) => (l.key === key ? { ...l, [side]: value, [other]: parseMoney(value) > 0 ? "" : l[other] } : l));
}

/** Lines ready for the API, or the first message the user must fix. */
export function payloadLines(lines: FormLine[]): { lines: JournalLineInput[] } | { error: string } {
  const filled = lines.filter((l) => l.accountId && (parseMoney(l.debit) > 0 || parseMoney(l.credit) > 0));
  if (filled.length < 2) return { error: fs.journals.minLines };
  const ids = filled.map((l) => l.accountId);
  if (new Set(ids).size !== ids.length) return { error: fs.journals.duplicateAccount };
  const t = lineTotals(filled);
  if (t.diff !== 0) return { error: fs.journals.notBalanced(formatRupiah(Math.abs(t.diff))) };
  if (t.debit <= 0 || t.credit <= 0) return { error: fs.journals.needBothSides };
  return {
    lines: filled.map((l) => {
      const isDebit = parseMoney(l.debit) > 0;
      return {
        accountId: l.accountId,
        entrySide: isDebit ? "DEBIT" : "CREDIT",
        amount: toMoneyString(parseMoney(isDebit ? l.debit : l.credit)),
        memo: l.memo.trim(),
      };
    }),
  };
}
