import { canAccess, type ErpAccess } from "@/components/shell/nav-config";

export type HrPerms = { read: boolean; write: boolean; approve: boolean; payroll: boolean };

export function hrPerms(access: ErpAccess): HrPerms {
  return {
    read: canAccess(access, "hr.read"),
    write: canAccess(access, "hr.write"),
    approve: canAccess(access, "hr.approve"),
    payroll: canAccess(access, "hr.payroll"),
  };
}

/** "2026-10" from a search param, defaulting to the current month. */
export function monthParam(v: string | undefined, now = new Date()): { ym: string; month: number; year: number } {
  const m = v?.match(/^(\d{4})-(\d{2})$/);
  const year = m ? Number(m[1]) : now.getFullYear();
  const month = m ? Number(m[2]) : now.getMonth() + 1;
  return { ym: `${year}-${String(month).padStart(2, "0")}`, month, year };
}

/** Today in Asia/Jakarta as "YYYY-MM-DD". */
export function todayWib(now = new Date()): string {
  return new Intl.DateTimeFormat("en-CA", { timeZone: "Asia/Jakarta" }).format(now);
}
