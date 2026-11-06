"use server";

import { revalidatePath } from "next/cache";
import { z } from "zod";
import { fail, type ActionResult } from "@/lib/action-result";
import { gatewayFetch } from "@/lib/erp/gateway";
import { runGateway, zodFieldErrors } from "@/features/erp/shared/action";
import { fs } from "./strings";
import type {
  Advance,
  ProjectTag,
  RecurringInvoice,
  ChargeType,
  FiscalPosition,
  PartyTaxProfile,
  TaxCode,
  DocumentItem,
  DocumentKind,
  FinanceDocument,
  Vendor,
  BudgetEntry,
  ClaimPayAccount,
  FixedAsset,
  MatchCandidate,
  RecurringJournal,
  StatementImportResult,
  StatementLine,
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
  isCreditCard: z.boolean(),
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
  tagId: z.string(),
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
  tagId: z.string(),
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
  taxIds: z.array(z.string()),
  tagId: z.string(),
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

/** The master codes a draft uses; the list rows do not carry them. */
export async function getInvoiceTaxIdsAction(kind: LedgerKind, invoiceId: string) {
  return runGateway(async () => (await gatewayFetch<{ taxIds?: string[] }>(`${base(kind)}/invoices/${id(invoiceId)}`)).taxIds ?? []);
}

export async function createCreditNoteAction(kind: LedgerKind, invoiceId: string, input: { noteDate: string; amount: string; reason: string; advanceAccountId: string }) {
  const res = await runGateway(() => gatewayFetch(`${F}/invoices/${id(invoiceId)}:credit-note`, json("POST", input)));
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
  adjustments: z.array(z.object({ kind: z.string(), accountId: z.string(), taxId: z.string(), amount: positive, memo: z.string().max(200) })),
  excessAmount: money,
  excessAccountId: z.string(),
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
  preferredCashAccountId: z.string().max(64),
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

// ─── Reconciliation ──────────────────────────

const RECON = `${ROOT}/reconciliation`;

async function reconAction<T>(path: string, body: unknown = {}): Promise<ActionResult<T>> {
  const res = await runGateway(() => gatewayFetch<T>(`${F}/recon/${path}`, json("POST", body)));
  revalidatePath(RECON);
  return res;
}

export async function rematchAction(accountId: string) {
  return reconAction<StatementImportResult>(`accounts/${id(accountId)}:rematch`);
}

export async function matchCandidatesAction(lineId: string) {
  return runGateway(() => gatewayFetch<{ candidates: MatchCandidate[] }>(`${F}/recon/lines/${id(lineId)}/candidates`));
}

export async function matchLineAction(lineId: string, kind: string, ref: string) {
  return reconAction<StatementLine>(`lines/${id(lineId)}:match`, { kind, ref });
}

export async function recordLineAction(lineId: string, counterAccountId: string, description: string) {
  return reconAction<StatementLine>(`lines/${id(lineId)}:record`, { counterAccountId, description });
}

export async function ignoreLineAction(lineId: string, note: string) {
  return reconAction<StatementLine>(`lines/${id(lineId)}:ignore`, { note });
}

export async function reopenLineAction(lineId: string) {
  return reconAction<StatementLine>(`lines/${id(lineId)}:reopen`);
}

export async function claimPayAccountsAction() {
  return runGateway(() => gatewayFetch<{ accounts: ClaimPayAccount[] }>(`${F}/reimbursements/pay-accounts`));
}

// ─── Schedules ───────────────────────────────

const SCHEDULES = `${ROOT}/schedules`;

async function scheduleAction<T>(path: string, method: string, body?: unknown): Promise<ActionResult<T>> {
  const res = await runGateway(() => gatewayFetch<T>(`${F}/${path}`, json(method, body)));
  revalidatePath(SCHEDULES);
  return res;
}

export type RecurringValues = {
  name: string;
  description: string;
  frequency: string;
  startDate: string;
  endDate: string;
  debitAccountId: string;
  creditAccountId: string;
  amount: string;
  tagId: string;
};

const recurringBody = ({ debitAccountId, creditAccountId, amount, ...rest }: RecurringValues) => ({
  ...rest,
  lines: [
    { accountId: debitAccountId, entrySide: "DEBIT", amount },
    { accountId: creditAccountId, entrySide: "CREDIT", amount },
  ],
});

export async function createRecurringAction(v: RecurringValues) {
  return scheduleAction<RecurringJournal>("recurring-journals", "POST", recurringBody(v));
}

export async function updateRecurringAction(recurringId: string, v: RecurringValues) {
  return scheduleAction<RecurringJournal>(`recurring-journals/${id(recurringId)}`, "PATCH", recurringBody(v));
}

export async function setRecurringActiveAction(recurringId: string, active: boolean) {
  return scheduleAction<RecurringJournal>(`recurring-journals/${id(recurringId)}:active`, "POST", { active });
}

export async function deleteRecurringAction(recurringId: string) {
  return scheduleAction(`recurring-journals/${id(recurringId)}`, "DELETE");
}

export type AssetValues = {
  name: string;
  acquiredOn: string;
  cost: string;
  salvage: string;
  lifeMonths: number;
  expenseAccountId: string;
  accumAccountId: string;
};

export async function createAssetAction(v: AssetValues) {
  return scheduleAction<FixedAsset>("fixed-assets", "POST", v);
}

export async function updateAssetAction(assetId: string, v: AssetValues) {
  return scheduleAction<FixedAsset>(`fixed-assets/${id(assetId)}`, "PATCH", v);
}

export async function setAssetActiveAction(assetId: string, active: boolean) {
  return scheduleAction<FixedAsset>(`fixed-assets/${id(assetId)}:active`, "POST", { active });
}

export async function deleteAssetAction(assetId: string) {
  return scheduleAction(`fixed-assets/${id(assetId)}`, "DELETE");
}

export async function runSchedulesAction() {
  const res = await scheduleAction<{ posted: number; failed: number }>("schedules:run", "POST", {});
  revalidatePath(`${ROOT}/journals`);
  return res;
}

export async function saveBudgetYearAction(year: number, entries: BudgetEntry[]) {
  const res = await runGateway(() => gatewayFetch(`${F}/budgets/${year}`, json("PUT", { entries })));
  revalidatePath(`${ROOT}/budgets`);
  revalidatePath(`${ROOT}/reports`);
  return res;
}

// ─── Vendors and documents ───────────────────

export type VendorValues = Omit<Vendor, "id"> & { id?: string };

export async function saveVendorAction(input: VendorValues) {
  const res = await runGateway(() => gatewayFetch<Vendor>(`${F}/vendors`, json("POST", input)));
  revalidatePath(`${ROOT}/vendors`);
  return res;
}

export async function deleteVendorAction(vendorId: string) {
  const res = await runGateway(() => gatewayFetch(`${F}/vendors/${id(vendorId)}`, json("DELETE")));
  revalidatePath(`${ROOT}/vendors`);
  return res;
}

export type DocumentValues = {
  id?: string;
  kind: DocumentKind;
  side: string;
  vendorId: string;
  partyName: string;
  picName: string;
  phone: string;
  email: string;
  address: string;
  title: string;
  issueDate: string;
  dueDate: string;
  externalNumber: string;
  notes: string;
  items: DocumentItem[];
  discountPercent: string;
  taxIds: string[];
  charges: { chargeId: string; value: string }[];
};

function revalidateDocuments() {
  revalidatePath(`${ROOT}/purchasing`);
  revalidatePath(`${ROOT}/documents`, "layout");
}

async function documentCall<T>(path: string, method: string, body?: unknown): Promise<ActionResult<T>> {
  const res = await runGateway(() => gatewayFetch<T>(`${F}/documents${path}`, json(method, body)));
  revalidateDocuments();
  return res;
}

export async function saveDocumentAction(input: DocumentValues) {
  return input.id ? documentCall<FinanceDocument>(`/${id(input.id)}`, "PATCH", input) : documentCall<FinanceDocument>("", "POST", input);
}

export async function deleteDocumentAction(documentId: string) {
  return documentCall(`/${id(documentId)}`, "DELETE");
}

export async function actOnDocumentAction(documentId: string, action: string, reason = "") {
  return documentCall<FinanceDocument>(`/${id(documentId)}:act`, "POST", { action, reason });
}

export async function convertDocumentAction(documentId: string, vendorId = "") {
  return documentCall<FinanceDocument>(`/${id(documentId)}:convert`, "POST", { vendorId });
}

export async function issueDocumentAction(documentId: string, accountId: string, dueDate: string) {
  const res = await documentCall<FinanceDocument>(`/${id(documentId)}:issue`, "POST", { accountId, dueDate });
  revalidatePath(`${ROOT}/payables`);
  revalidatePath(`${ROOT}/receivables`);
  return res;
}

export async function saveTaxCodeAction(input: TaxCode) {
  const res = await runGateway(() => gatewayFetch<TaxCode>(`${F}/tax-codes`, json("POST", input)));
  revalidatePath(`${ROOT}/tax-setup`);
  return res;
}

async function taxMasterAction<T>(path: string, method: string, body?: unknown): Promise<ActionResult<T>> {
  const res = await runGateway(() => gatewayFetch<T>(`${F}/${path}`, json(method, body)));
  revalidatePath(`${ROOT}/tax-setup`);
  return res;
}

export async function deleteTaxCodeAction(taxId: string) {
  return taxMasterAction(`tax-codes/${id(taxId)}`, "DELETE");
}

export async function deleteChargeTypeAction(chargeId: string) {
  return taxMasterAction(`charge-types/${id(chargeId)}`, "DELETE");
}

export async function saveFiscalPositionAction(input: FiscalPosition) {
  return taxMasterAction<FiscalPosition>("fiscal-positions", "POST", input);
}

export async function deleteFiscalPositionAction(positionId: string) {
  return taxMasterAction(`fiscal-positions/${id(positionId)}`, "DELETE");
}

export async function savePartyProfileAction(input: PartyTaxProfile) {
  return taxMasterAction<PartyTaxProfile>("party-tax-profiles", "POST", input);
}

export async function deletePartyProfileAction(profileId: string) {
  return taxMasterAction(`party-tax-profiles/${id(profileId)}`, "DELETE");
}

export async function saveChargeTypeAction(input: ChargeType) {
  const res = await runGateway(() => gatewayFetch<ChargeType>(`${F}/charge-types`, json("POST", input)));
  revalidatePath(`${ROOT}/tax-setup`);
  return res;
}

// ─── Projects, recurring invoices, advances ──────────────────

export async function saveProjectTagAction(input: ProjectTag) {
  const res = await runGateway(() => gatewayFetch<ProjectTag>(`${F}/project-tags`, json("POST", input)));
  revalidatePath(`${ROOT}/projects`);
  return res;
}

export async function deleteProjectTagAction(tagId: string) {
  const res = await runGateway(() => gatewayFetch(`${F}/project-tags/${id(tagId)}`, json("DELETE")));
  revalidatePath(`${ROOT}/projects`);
  return res;
}

export type RecurringInvoiceValues = Pick<RecurringInvoice, "kind" | "name" | "partyName" | "clientId" | "accountId" | "tagId" | "description" | "subtotal" | "taxAmount" | "dueDays" | "frequency" | "endDate" | "autoPost"> & {
  taxIds: string[];
  startDate: string;
};

export async function saveRecurringInvoiceAction(invoiceId: string | null, v: RecurringInvoiceValues) {
  return invoiceId ? scheduleAction<RecurringInvoice>(`recurring-invoices/${id(invoiceId)}`, "PATCH", v) : scheduleAction<RecurringInvoice>("recurring-invoices", "POST", v);
}

export async function setRecurringInvoiceActiveAction(invoiceId: string, active: boolean) {
  return scheduleAction<RecurringInvoice>(`recurring-invoices/${id(invoiceId)}:active`, "POST", { active });
}

export async function deleteRecurringInvoiceAction(invoiceId: string) {
  return scheduleAction(`recurring-invoices/${id(invoiceId)}`, "DELETE");
}

export async function createAdvanceAction(input: { kind: LedgerKind; advanceDate: string; partyName: string; amount: string; cashAccountId: string; advanceAccountId: string; reference: string; notes: string }) {
  const res = await runGateway(() => gatewayFetch<Advance>(`${F}/advances`, json("POST", input)));
  revalidatePath(pagePath(input.kind));
  return res;
}

export async function applyAdvanceAction(kind: LedgerKind, advanceId: string, input: { invoiceId: string; amount: string; date: string }) {
  const res = await runGateway(() => gatewayFetch(`${F}/advances/${id(advanceId)}:apply`, json("POST", input)));
  revalidatePath(pagePath(kind));
  return res;
}

export async function voidAdvanceAction(kind: LedgerKind, advanceId: string, reason: string) {
  const res = await runGateway(() => gatewayFetch<Advance>(`${F}/advances/${id(advanceId)}:void`, json("POST", { reason })));
  revalidatePath(pagePath(kind));
  return res;
}

export async function refundAdvanceAction(kind: LedgerKind, advanceId: string, input: { refundDate: string; amount: string; cashAccountId: string; reason: string }) {
  const res = await runGateway(() => gatewayFetch<Advance>(`${F}/advances/${id(advanceId)}:refund`, json("POST", input)));
  revalidatePath(pagePath(kind));
  return res;
}

export async function voidAdvanceRefundAction(kind: LedgerKind, refundId: string, reason: string) {
  const res = await runGateway(() => gatewayFetch<Advance>(`${F}/advance-refunds/${id(refundId)}:void`, json("POST", { reason })));
  revalidatePath(pagePath(kind));
  return res;
}

const periodEndPath = "/erp/finance/period-end";

async function periodEnd<T>(call: () => Promise<T>) {
  const res = await runGateway(call);
  revalidatePath(periodEndPath);
  return res;
}

export async function createDeferralAction(input: { kind: string; name: string; partyName: string; reference: string; deferredAccountId: string; targetAccountId: string; startDate: string; months: number; amount: string }) {
  return periodEnd(() => gatewayFetch(`${F}/deferrals`, json("POST", input)));
}

export async function setDeferralActiveAction(deferralId: string, active: boolean) {
  return periodEnd(() => gatewayFetch(`${F}/deferrals/${id(deferralId)}:active`, json("POST", { active })));
}

export async function deleteDeferralAction(deferralId: string) {
  return periodEnd(() => gatewayFetch(`${F}/deferrals/${id(deferralId)}`, json("DELETE")));
}

export async function remitTaxAction(input: { period: string; remitDate: string; accountId: string; cashAccountId: string; amount: string; ntpn: string; notes: string }) {
  return periodEnd(() => gatewayFetch(`${F}/tax-remittances`, json("POST", input)));
}

export async function voidTaxRemittanceAction(remittanceId: string, reason: string) {
  return periodEnd(() => gatewayFetch(`${F}/tax-remittances/${id(remittanceId)}:void`, json("POST", { reason })));
}

export async function closeVatAction(period: string) {
  return periodEnd(() => gatewayFetch<{ offset: string; payable: string; compensation: string }>(`${F}/vat-closings`, json("POST", { period })));
}

export async function setSeparateApproverAction(separateApprover: boolean) {
  return periodEnd(() => gatewayFetch(`${F}/control-settings`, json("PUT", { separateApprover })));
}

export async function writeOffInvoiceAction(invoiceId: string, input: { writeOffDate: string; amount: string; expenseAccountId: string; reason: string }) {
  const res = await runGateway(() => gatewayFetch(`${F}/invoices/${id(invoiceId)}:write-off`, json("POST", input)));
  revalidatePath(pagePath("AR"));
  return res;
}

export async function reverseJournalAction(entryId: string, reason: string) {
  const res = await runGateway(() => gatewayFetch(`${F}/journal-entries/${id(entryId)}:reverse`, json("POST", { reason })));
  revalidatePath("/erp/finance/journals", "layout");
  return res;
}

export async function disposeAssetAction(assetId: string, input: { disposalDate: string; assetAccountId: string; cashAccountId: string; gainLossAccountId: string; proceeds: string; reason: string }) {
  return scheduleAction<FixedAsset>(`fixed-assets/${id(assetId)}:dispose`, "POST", input);
}

const poPath = "/erp/finance/po-matching";

export async function receiveGoodsAction(input: { documentId: string; receiptDate: string; amount: string; notes: string }) {
  const res = await runGateway(() => gatewayFetch(`${F}/goods-receipts`, json("POST", input)));
  revalidatePath(poPath);
  return res;
}

export async function voidGoodsReceiptAction(receiptId: string, reason: string) {
  const res = await runGateway(() => gatewayFetch(`${F}/goods-receipts/${id(receiptId)}:void`, json("POST", { reason })));
  revalidatePath(poPath);
  return res;
}

export async function matchInvoiceToPoAction(invoiceId: string, documentId: string) {
  const res = await runGateway(() => gatewayFetch(`${F}/ap/invoices/${id(invoiceId)}:match-po`, json("POST", { documentId })));
  revalidatePath(poPath);
  return res;
}

const projectsPath = "/erp/finance/projects";

export async function createProjectAllocationAction(input: { tagId: string; month: string; kind: string; amount: string; notes: string }) {
  const res = await runGateway(() => gatewayFetch(`${F}/project-allocations`, json("POST", input)));
  revalidatePath(projectsPath);
  return res;
}

export async function deleteProjectAllocationAction(allocationId: string) {
  const res = await runGateway(() => gatewayFetch(`${F}/project-allocations/${id(allocationId)}`, json("DELETE")));
  revalidatePath(projectsPath);
  return res;
}
