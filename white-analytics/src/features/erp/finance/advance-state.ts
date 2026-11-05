import { parseMoney } from "@/features/erp/shared/money";
import type { Advance } from "./types";

export function advanceState(a: Advance) {
  if (a.status === "VOID") return "void";
  return parseMoney(a.remaining) > 0 ? "open" : "used";
}
