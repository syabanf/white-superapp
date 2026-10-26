import { canAccess, type ErpAccess } from "@/components/shell/nav-config";

export type FinancePerms = { canWrite: boolean; canApprove: boolean };

/** finance.write covers drafts and cash-bank; finance.approve covers post, close, void, pay and policies. */
export function financePerms(access: ErpAccess): FinancePerms {
  return { canWrite: canAccess(access, "finance.write"), canApprove: canAccess(access, "finance.approve") };
}
