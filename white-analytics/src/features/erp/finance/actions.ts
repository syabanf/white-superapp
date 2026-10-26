"use server";

import { revalidatePath } from "next/cache";
import { z } from "zod";
import { fail, type ActionResult } from "@/lib/action-result";
import { gatewayFetch } from "@/lib/erp/gateway";
import { runGateway, zodFieldErrors } from "@/features/erp/shared/action";
import { fs } from "./strings";
import type {
  Account,
  AccountType,
  Claim,
  FinancialInsight,
  FiscalCoverage,
  FiscalPeriod,
  FiscalYear,
  ImportAccountsResponse,
  InvoiceResult,
  JournalEntry,
  JournalMapping,
  LedgerKind,
  PaymentResult,
  PeriodClosePreview,
  PostingLog,
  ReimbursementBatch,
  ReimbursementPolicy,
  SaveBeginningBalanceResult,
} from "./types";

const F = "/v1/finance";
const ROOT = "/erp/finance";
const id = (v: string) => encodeURIComponent(v);
const json = (method: string, body?: unknown): RequestInit => ({
  method,
  body: body === undefined ? undefined : JSON.stringify(body),
});

function invalid<T>(err: z.ZodError): ActionResult<T> {
  const fieldErrors = zodFieldErrors(err);
  const first = Object.values(fieldErrors)[0]?.[0];
  return fail(first ?? "Periksa kembali data yang dimasukkan.", "VALIDATION", fieldErrors);
}

const money = z.string().regex(/^-?\d+(\.\d{1,2})?$/, "Format jumlah tidak valid.");
const day = z.string().regex(/^\d{4}-\d{2}-\d{2}$/, fs.required);
const side = z.enum(["DEBIT", "CREDIT"]);

// ── Account types ────────────────────────────────────────────

const accountTypeSchema = z.object({
  code: z.string().trim().min(1, fs.accounts.codeRequired).max(20),
  name: z.string().trim().min(1, fs.accounts.nameRequired).max(80),
  normalBalance: side,
  sortOrder: z.number().int().min(0).max(9999),
  isActive: z.boolean(),
});

export async function saveAccountTypeAction(
  typeId: string | null,
  input: z.input<typeof accountTypeSchema>,
): Promise<ActionResult<AccountType>> {
  const p = accountTypeSchema.safeParse(input);
  if (!p.success) return invalid(p.error);
  const res = await runGateway(() =>
    typeId
      ? gatewayFetch<AccountType>(`${F}/account-types/${id(typeId)}`, json("PATCH", p.data))
      : gatewayFetch<AccountType>(`${F}/account-types`, json("POST", p.data)),
  );
  revalidatePath(`${ROOT}/accounts`);
  return res;
}

export async function deleteAccountTypeAction(typeId: string) {
  const res = await runGateway(() => gatewayFetch(`${F}/account-types/${id(typeId)}`, json("DELETE")));
  revalidatePath(`${ROOT}/accounts`);
  return res;
}

// ── Accounts ─────────────────────────────────────────────────

const accountSchema = z.object({
  code: z.string().trim().min(1, fs.accounts.codeRequired).max(20),
  name: z.string().trim().min(1, fs.accounts.nameRequired).max(160),
  parentId: z.string().nullable().optional(),
  accountTypeId: z.string().min(1, fs.accounts.typeRequired),
  isContra: z.boolean(),
  isCashBank: z.boolean(),
  cashFlowCategory: z.enum(["", "OPERATING", "INVESTING", "FINANCING", "NON_CASH"]),
  description: z.string().max(500),
  isActive: z.boolean(),
});

export async function saveAccountAction(
  accountId: string | null,
  input: z.input<typeof accountSchema>,
): Promise<ActionResult<Account>> {
  const p = accountSchema.safeParse(input);
  if (!p.success) return invalid(p.error);
  const body = { ...p.data, parentId: p.data.parentId || undefined };
  const res = await runGateway(() =>
    accountId
      ? gatewayFetch<Account>(`${F}/accounts/${id(accountId)}`, json("PATCH", body))
      : gatewayFetch<Account>(`${F}/accounts`, json("POST", body)),
  );
  revalidatePath(`${ROOT}/accounts`);
  return res;
}

export async function deleteAccountAction(accountId: string) {
  const res = await runGateway(() => gatewayFetch(`${F}/accounts/${id(accountId)}`, json("DELETE")));
  revalidatePath(`${ROOT}/accounts`);
  return res;
}

/** XLSX import: the client sends the file as base64 (the RPC takes bytes in JSON). */
export async function importAccountsAction(
  fileBase64: string,
  mode: "preview" | "commit",
): Promise<ActionResult<ImportAccountsResponse>> {
  if (!fileBase64 || fileBase64.length > 7_000_000) return fail(fs.accounts.import.tooLarge, "VALIDATION");
  const res = await runGateway(() =>
    gatewayFetch<ImportAccountsResponse>(`${F}/accounts:import`, json("POST", { file: fileBase64, mode })),
  );
  if (mode === "commit") revalidatePath(`${ROOT}/accounts`);
  return res;
}

// ── Journal mappings ─────────────────────────────────────────

const mappingSchema = z.object({
  eventCode: z.string().min(1, fs.accounts.mappings.eventRequired),
  name: z.string().trim().min(1, fs.accounts.nameRequired).max(120),
  description: z.string().max(500),
  module: z.string(),
  isActive: z.boolean(),
  lines: z
    .array(
      z.object({
        entrySide: side,
        lineRole: z.string().trim().max(60),
        accountId: z.string().nullable().optional(),
        amountSource: z.string().min(1),
        sortOrder: z.number().int().optional(),
        isRequired: z.boolean().optional(),
      }),
    )
    .min(1, fs.accounts.mappings.noLines),
});

export async function saveJournalMappingAction(
  mappingId: string | null,
  input: z.input<typeof mappingSchema>,
): Promise<ActionResult<JournalMapping>> {
  const p = mappingSchema.safeParse(input);
  if (!p.success) return invalid(p.error);
  const body = { ...p.data, lines: p.data.lines.map((l) => ({ ...l, accountId: l.accountId || undefined })) };
  const res = await runGateway(() =>
    mappingId
      ? gatewayFetch<JournalMapping>(`${F}/journal-mappings/${id(mappingId)}`, json("PATCH", body))
      : gatewayFetch<JournalMapping>(`${F}/journal-mappings`, json("POST", body)),
  );
  revalidatePath(`${ROOT}/accounts`);
  return res;
}

export async function deleteJournalMappingAction(mappingId: string) {
  const res = await runGateway(() => gatewayFetch(`${F}/journal-mappings/${id(mappingId)}`, json("DELETE")));
  revalidatePath(`${ROOT}/accounts`);
  return res;
}

// ── Fiscal ───────────────────────────────────────────────────

const fiscalYearSchema = z
  .object({
    code: z.string().trim().min(1, fs.fiscal.codeRequired).max(20),
    name: z.string().trim().min(1, fs.fiscal.nameRequired).max(80),
    startDate: day,
    endDate: day,
    isActive: z.boolean(),
  })
  .refine((v) => v.endDate > v.startDate, { message: fs.fiscal.dateOrder, path: ["endDate"] });

export async function saveFiscalYearAction(
  yearId: string | null,
  input: z.input<typeof fiscalYearSchema>,
): Promise<ActionResult<FiscalYear>> {
  const p = fiscalYearSchema.safeParse(input);
  if (!p.success) return invalid(p.error);
  const res = await runGateway(async () => {
    if (!yearId) return gatewayFetch<FiscalYear>(`${F}/fiscal-years`, json("POST", { ...p.data, periods: [] }));
    // Keep the existing periods; the update replaces them when the list is sent.
    const current = await gatewayFetch<FiscalYear>(`${F}/fiscal-years/${id(yearId)}`);
    const periods = current.periods.map((pr) => ({
      periodNo: pr.periodNo,
      name: pr.name,
      startDate: pr.startDate,
      endDate: pr.endDate,
      status: pr.status,
    }));
    return gatewayFetch<FiscalYear>(`${F}/fiscal-years/${id(yearId)}`, json("PATCH", { ...p.data, periods }));
  });
  revalidatePath(`${ROOT}/fiscal`);
  return res;
}

export async function deleteFiscalYearAction(yearId: string) {
  const res = await runGateway(() => gatewayFetch(`${F}/fiscal-years/${id(yearId)}`, json("DELETE")));
  revalidatePath(`${ROOT}/fiscal`);
  return res;
}

export async function openPeriodAction(periodId: string, closePrevious: boolean) {
  const res = await runGateway(() =>
    gatewayFetch<FiscalPeriod>(`${F}/fiscal-periods/${id(periodId)}:open`, json("POST", { closePrevious })),
  );
  revalidatePath(ROOT, "layout");
  return res;
}

export async function closePeriodPreviewAction(periodId: string) {
  return runGateway(() => gatewayFetch<PeriodClosePreview>(`${F}/fiscal-periods/${id(periodId)}/close-preview`));
}

export async function closePeriodAction(periodId: string) {
  const res = await runGateway(() =>
    gatewayFetch<FiscalPeriod>(`${F}/fiscal-periods/${id(periodId)}:close`, json("POST")),
  );
  revalidatePath(ROOT, "layout");
  return res;
}

export async function fiscalCoverageAction(date: string) {
  if (!/^\d{4}-\d{2}-\d{2}$/.test(date)) return fail<FiscalCoverage>(fs.journals.dateRequired, "VALIDATION");
  return runGateway(() => gatewayFetch<FiscalCoverage>(`${F}/fiscal-coverage?date=${id(date)}`));
}

const bbSchema = z.object({
  lines: z.array(z.object({ accountId: z.string().min(1), entrySide: side, amount: money, memo: z.string() })),
  post: z.boolean(),
});

export async function saveBeginningBalanceAction(
  yearId: string,
  input: z.input<typeof bbSchema>,
): Promise<ActionResult<SaveBeginningBalanceResult>> {
  const p = bbSchema.safeParse(input);
  if (!p.success) return invalid(p.error);
  const res = await runGateway(() =>
    gatewayFetch<SaveBeginningBalanceResult>(`${F}/fiscal-years/${id(yearId)}/beginning-balance`, json("PUT", p.data)),
  );
  revalidatePath(`${ROOT}/fiscal`, "layout");
  return res;
}

// ── Journal entries ──────────────────────────────────────────

const journalSchema = z.object({
  entryDate: day,
  description: z.string().trim().min(1, fs.journals.descriptionRequired).max(500),
  isRecon: z.boolean(),
  lines: z
    .array(z.object({ accountId: z.string().min(1), entrySide: side, amount: money, memo: z.string().max(300) }))
    .min(2, fs.journals.minLines),
  post: z.boolean(),
});

export async function saveJournalEntryAction(
  entryId: string | null,
  input: z.input<typeof journalSchema>,
): Promise<ActionResult<JournalEntry>> {
  const p = journalSchema.safeParse(input);
  if (!p.success) return invalid(p.error);
  const res = await runGateway(() =>
    entryId
      ? gatewayFetch<JournalEntry>(`${F}/journal-entries/${id(entryId)}`, json("PATCH", p.data))
      : gatewayFetch<JournalEntry>(`${F}/journal-entries`, json("POST", p.data)),
  );
  revalidatePath(`${ROOT}/journals`, "layout");
  return res;
}

export async function deleteJournalEntryAction(entryId: string) {
  const res = await runGateway(() => gatewayFetch(`${F}/journal-entries/${id(entryId)}`, json("DELETE")));
  revalidatePath(`${ROOT}/journals`, "layout");
  return res;
}

export async function postJournalEntryAction(entryId: string) {
  const res = await runGateway(() =>
    gatewayFetch<JournalEntry>(`${F}/journal-entries/${id(entryId)}:post`, json("POST")),
  );
  revalidatePath(`${ROOT}/journals`, "layout");
  return res;
}

export async function retryPostingAction(logId: string) {
  const res = await runGateway(() => gatewayFetch<PostingLog>(`${F}/posting-logs/${id(logId)}:retry`, json("POST")));
  revalidatePath(`${ROOT}/journals`, "layout");
  return res;
}

// ── Cash & bank ──────────────────────────────────────────────

const positive = money.refine((v) => Number(v) > 0, fs.cash.amountRequired);

const movementSchema = z.object({
  kind: z.enum(["CASH_IN", "CASH_OUT"]),
  entryDate: day,
  amount: positive,
  cashAccountId: z.string().min(1, fs.cash.accountRequired),
  offsetAccountId: z.string().min(1, fs.cash.accountRequired),
  description: z.string().trim().min(1, fs.journals.descriptionRequired).max(500),
  memo: z.string().max(300),
});

export async function createCashMovementAction(input: z.input<typeof movementSchema>) {
  const p = movementSchema.safeParse(input);
  if (!p.success) return invalid(p.error);
  const res = await runGateway(() => gatewayFetch(`${F}/cash-bank/movements`, json("POST", p.data)));
  revalidatePath(`${ROOT}/cash-bank`, "layout");
  return res;
}

const transferSchema = z
  .object({
    entryDate: day,
    amount: positive,
    fromAccountId: z.string().min(1, fs.cash.accountRequired),
    toAccountId: z.string().min(1, fs.cash.accountRequired),
    description: z.string().trim().min(1, fs.journals.descriptionRequired).max(500),
    memo: z.string().max(300),
  })
  .refine((v) => v.fromAccountId !== v.toAccountId, { message: fs.cash.sameAccount, path: ["toAccountId"] });

export async function createCashTransferAction(input: z.input<typeof transferSchema>) {
  const p = transferSchema.safeParse(input);
  if (!p.success) return invalid(p.error);
  const res = await runGateway(() => gatewayFetch(`${F}/cash-bank/transfers`, json("POST", p.data)));
  revalidatePath(`${ROOT}/cash-bank`, "layout");
  return res;
}

// ── AP / AR ──────────────────────────────────────────────────

const base = (kind: LedgerKind) => `${F}/${kind === "AP" ? "ap" : "ar"}`;
const pagePath = (kind: LedgerKind) => `${ROOT}/${kind === "AP" ? "payables" : "receivables"}`;

const invoiceSchema = z.object({
  invoiceDate: day,
  dueDate: day,
  partyName: z.string().trim().min(1, fs.invoices.partyRequired).max(160),
  reference: z.string().trim().max(80),
  accountId: z.string(),
  subtotal: money.refine((v) => Number(v) > 0, fs.invoices.subtotalRequired),
  taxAmount: money,
  description: z.string().max(500),
  clientId: z.string(),
});

export async function saveInvoiceAction(kind: LedgerKind, invoiceId: string | null, input: z.input<typeof invoiceSchema>) {
  const p = invoiceSchema.safeParse(input);
  if (!p.success) return invalid(p.error);
  const res = await runGateway(() =>
    invoiceId
      ? gatewayFetch(`${base(kind)}/invoices/${id(invoiceId)}`, json("PATCH", p.data))
      : gatewayFetch(`${base(kind)}/invoices`, json("POST", p.data)),
  );
  revalidatePath(pagePath(kind));
  return res;
}

export async function deleteInvoiceAction(kind: LedgerKind, invoiceId: string) {
  const res = await runGateway(() => gatewayFetch(`${base(kind)}/invoices/${id(invoiceId)}`, json("DELETE")));
  revalidatePath(pagePath(kind));
  return res;
}

export async function postInvoiceAction(kind: LedgerKind, invoiceId: string) {
  const res = await runGateway(() =>
    gatewayFetch<InvoiceResult>(`${base(kind)}/invoices/${id(invoiceId)}:post`, json("POST")),
  );
  revalidatePath(pagePath(kind));
  return res;
}

const paymentSchema = z.object({
  paymentDate: day,
  method: z.string().min(1),
  cashAccountId: z.string(),
  referenceNumber: z.string().max(80),
  notes: z.string().max(500),
  allocations: z
    .array(z.object({ invoiceId: z.string().min(1), amount: positive }))
    .min(1, fs.invoices.allocationRequired),
});

export async function createPaymentAction(
  kind: LedgerKind,
  input: z.input<typeof paymentSchema>,
): Promise<ActionResult<PaymentResult>> {
  const p = paymentSchema.safeParse(input);
  if (!p.success) return invalid(p.error);
  const res = await runGateway(() =>
    gatewayFetch<PaymentResult>(`${base(kind)}/${kind === "AP" ? "payments" : "receipts"}`, json("POST", p.data)),
  );
  revalidatePath(pagePath(kind));
  return res;
}

export async function voidPaymentAction(kind: LedgerKind, paymentId: string, reason: string) {
  if (!reason.trim()) return fail<PaymentResult>(fs.invoices.voidReasonRequired, "VALIDATION");
  const res = await runGateway(() =>
    gatewayFetch<PaymentResult>(
      `${base(kind)}/${kind === "AP" ? "payments" : "receipts"}/${id(paymentId)}:void`,
      json("POST", { reason: reason.trim() }),
    ),
  );
  revalidatePath(pagePath(kind));
  return res;
}

// ── Reimbursements ───────────────────────────────────────────

const claimSchema = z.object({
  category: z.enum(["TRANSPORT", "MAKAN", "AKOMODASI", "OPERASIONAL_PROYEK", "LAINNYA"], {
    message: fs.reimburse.categoryRequired,
  }),
  transactionDate: day,
  amount: money.refine((v) => Number(v) > 0, fs.reimburse.amountRequired),
  description: z.string().trim().min(1, fs.reimburse.descriptionRequired).max(500),
  clientId: z.string().max(64),
  project: z.string().max(120),
});

function revalidateClaims(claimId?: string) {
  revalidatePath("/erp/me/reimbursements", "layout");
  revalidatePath(`${ROOT}/reimbursements`, "layout");
  if (claimId) revalidatePath(`/erp/me/reimbursements/${claimId}`);
}

export async function saveClaimAction(claimId: string | null, input: z.input<typeof claimSchema>) {
  const p = claimSchema.safeParse(input);
  if (!p.success) return invalid<Claim>(p.error);
  const res = await runGateway(() =>
    claimId
      ? gatewayFetch<Claim>(`${F}/reimbursements/${id(claimId)}`, json("PATCH", p.data))
      : gatewayFetch<Claim>(`${F}/reimbursements`, json("POST", p.data)),
  );
  revalidateClaims(claimId ?? undefined);
  return res;
}

export async function deleteReceiptAction(claimId: string, receiptId: string) {
  const res = await runGateway(() =>
    gatewayFetch<Claim>(`${F}/reimbursements/${id(claimId)}/receipts/${id(receiptId)}`, json("DELETE")),
  );
  revalidateClaims(claimId);
  return res;
}

export async function claimTransitionAction(claimId: string, action: "submit" | "cancel") {
  const res = await runGateway(() => gatewayFetch<Claim>(`${F}/reimbursements/${id(claimId)}:${action}`, json("POST")));
  revalidateClaims(claimId);
  return res;
}

export async function approveClaimAction(claimId: string, note: string) {
  const res = await runGateway(() =>
    gatewayFetch<Claim>(`${F}/reimbursements/${id(claimId)}:approve`, json("POST", { note: note.trim() })),
  );
  revalidateClaims(claimId);
  return res;
}

export async function rejectClaimAction(claimId: string, reason: string) {
  if (!reason.trim()) return fail<Claim>(fs.reimburse.rejectReasonRequired, "VALIDATION");
  const res = await runGateway(() =>
    gatewayFetch<Claim>(`${F}/reimbursements/${id(claimId)}:reject`, json("POST", { reason: reason.trim() })),
  );
  revalidateClaims(claimId);
  return res;
}

const paySchema = z.object({
  claimIds: z.array(z.string().min(1)).min(1, fs.reimburse.selectToPay),
  cashAccountId: z.string(),
  paidAt: day,
  notes: z.string().max(500),
});

export async function payClaimsAction(input: z.input<typeof paySchema>) {
  const p = paySchema.safeParse(input);
  if (!p.success) return invalid<ReimbursementBatch>(p.error);
  const res = await runGateway(() =>
    gatewayFetch<ReimbursementBatch>(`${F}/reimbursements:pay`, json("POST", p.data)),
  );
  revalidateClaims();
  return res;
}

const policySchema = z.object({
  category: z.enum(["TRANSPORT", "MAKAN", "AKOMODASI", "OPERASIONAL_PROYEK", "LAINNYA"]),
  monthlyLimit: z.union([z.literal(""), money]),
  perClaimLimit: z.union([z.literal(""), money]),
  expenseAccountId: z.string(),
  isActive: z.boolean(),
  autoApprove: z.boolean(),
});

export async function updatePolicyAction(input: z.input<typeof policySchema>) {
  const p = policySchema.safeParse(input);
  if (!p.success) return invalid<ReimbursementPolicy>(p.error);
  const res = await runGateway(() =>
    gatewayFetch<ReimbursementPolicy>(`${F}/reimbursement-policies/${p.data.category}`, json("PUT", p.data)),
  );
  revalidatePath(`${ROOT}/reimbursements`);
  return res;
}

// ── Analysis ─────────────────────────────────────────────────

export async function generateInsightAction(from: string, to: string, language: "id" | "en") {
  const res = await runGateway(() =>
    gatewayFetch<FinancialInsight>(`${F}/analysis/insights`, json("POST", { from, to, language })),
  );
  revalidatePath(`${ROOT}/analysis`);
  return res;
}
