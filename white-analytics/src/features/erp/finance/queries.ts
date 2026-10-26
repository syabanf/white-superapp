import "server-only";
import { gatewayFetch } from "@/lib/erp/gateway";
import { qs } from "@/features/erp/shared/load";
import type {
  Account,
  AccountType,
  Aging,
  BalanceSheet,
  BeginningBalance,
  CashBankAccount,
  CashFlow,
  CashMovement,
  CashTransfer,
  Claim,
  ClaimsResponse,
  Dashboard,
  FinancialAnalysis,
  FinancialInsight,
  FiscalYear,
  GeneralLedgerAccount,
  IncomeStatement,
  Invoice,
  JournalEntry,
  JournalEventsResponse,
  JournalMapping,
  Ledger,
  LedgerKind,
  MonthlySummary,
  Payment,
  PostableAccount,
  PostingLog,
  ReimbursementBatch,
  ReimbursementPolicy,
  SubsidiaryLedger,
  SubsidiaryParty,
  TrialBalance,
} from "./types";

const F = "/v1/finance";
type Q = Record<string, string | number | boolean | undefined | null>;

// ── Accounts ─────────────────────────────────────────────────

export async function listAccountTypes(includeInactive = true) {
  const r = await gatewayFetch<{ accountTypes: AccountType[] }>(`${F}/account-types${qs({ includeInactive })}`);
  return r.accountTypes;
}

export async function listAccounts(q: Q = {}) {
  const r = await gatewayFetch<{ accounts: Account[] }>(`${F}/accounts${qs(q)}`);
  return r.accounts;
}

export async function listPostableAccounts() {
  const r = await gatewayFetch<{ accounts: PostableAccount[] }>(`${F}/accounts:postable`);
  return r.accounts;
}

export function listJournalEvents() {
  return gatewayFetch<JournalEventsResponse>(`${F}/journal-events`);
}

export async function listJournalMappings(q: Q = {}) {
  const r = await gatewayFetch<{ mappings: JournalMapping[] }>(`${F}/journal-mappings${qs(q)}`);
  return r.mappings;
}

// ── Fiscal ───────────────────────────────────────────────────

export async function listFiscalYears() {
  const r = await gatewayFetch<{ fiscalYears: FiscalYear[] }>(`${F}/fiscal-years`);
  return r.fiscalYears;
}

export function getFiscalYear(id: string) {
  return gatewayFetch<FiscalYear>(`${F}/fiscal-years/${encodeURIComponent(id)}`);
}

export function getBeginningBalance(id: string) {
  return gatewayFetch<BeginningBalance>(`${F}/fiscal-years/${encodeURIComponent(id)}/beginning-balance`);
}

// ── Journals ─────────────────────────────────────────────────

export function listJournalEntries(q: Q) {
  return gatewayFetch<{ entries: JournalEntry[]; total: number }>(`${F}/journal-entries${qs(q)}`);
}

export function getJournalEntry(id: string) {
  return gatewayFetch<JournalEntry>(`${F}/journal-entries/${encodeURIComponent(id)}`);
}

export async function listPostingLogs(q: Q = {}) {
  const r = await gatewayFetch<{ logs: PostingLog[] }>(`${F}/posting-logs${qs({ limit: 200, ...q })}`);
  return r.logs;
}

// ── Cash & bank ──────────────────────────────────────────────

export function listCashBankAccounts() {
  return gatewayFetch<{ accounts: CashBankAccount[]; totalBalance: string }>(`${F}/cash-bank`);
}

export function getCashBankLedger(accountId: string, q: Q) {
  return gatewayFetch<Ledger>(`${F}/cash-bank/${encodeURIComponent(accountId)}/ledger${qs(q)}`);
}

export function listCashMovements(q: Q) {
  return gatewayFetch<{ movements: CashMovement[]; total: number }>(`${F}/cash-bank/movements${qs(q)}`);
}

export function listCashTransfers(q: Q) {
  return gatewayFetch<{ transfers: CashTransfer[]; total: number }>(`${F}/cash-bank/transfers${qs(q)}`);
}

// ── AP / AR ──────────────────────────────────────────────────

const ledgerBase = (kind: LedgerKind) => `${F}/${kind === "AP" ? "ap" : "ar"}`;

export function listInvoices(kind: LedgerKind, q: Q) {
  return gatewayFetch<{ invoices: Invoice[]; total: number }>(`${ledgerBase(kind)}/invoices${qs(q)}`);
}

/** Posted invoices with an outstanding balance, for payment allocation. */
export async function listOpenInvoices(kind: LedgerKind) {
  const r = await gatewayFetch<{ invoices: Invoice[] }>(`${ledgerBase(kind)}/${kind === "AP" ? "payable" : "receivable"}`);
  return r.invoices;
}

export function getAging(kind: LedgerKind, asOf?: string) {
  return gatewayFetch<Aging>(`${ledgerBase(kind)}/aging${qs({ asOf })}`);
}

export function listPayments(kind: LedgerKind, q: Q) {
  return gatewayFetch<{ payments: Payment[]; total: number }>(
    `${ledgerBase(kind)}/${kind === "AP" ? "payments" : "receipts"}${qs(q)}`,
  );
}

// ── Reports ──────────────────────────────────────────────────

export function getDashboard(q: Q) {
  return gatewayFetch<Dashboard>(`${F}/dashboard${qs(q)}`);
}

export function getTrialBalance(asOf?: string) {
  return gatewayFetch<TrialBalance>(`${F}/reports/trial-balance${qs({ asOf })}`);
}

export function getBalanceSheet(asOf?: string) {
  return gatewayFetch<BalanceSheet>(`${F}/reports/balance-sheet${qs({ asOf })}`);
}

export function getIncomeStatement(dateFrom?: string, dateTo?: string) {
  return gatewayFetch<IncomeStatement>(`${F}/reports/income-statement${qs({ dateFrom, dateTo })}`);
}

export function getCashFlow(dateFrom?: string, dateTo?: string) {
  return gatewayFetch<CashFlow>(`${F}/reports/cash-flow${qs({ dateFrom, dateTo })}`);
}

export async function listGeneralLedgerAccounts(asOf?: string) {
  const r = await gatewayFetch<{ accounts: GeneralLedgerAccount[] }>(
    `${F}/reports/general-ledger-accounts${qs({ asOf })}`,
  );
  return r.accounts;
}

export function getGeneralLedger(accountId: string, dateFrom?: string, dateTo?: string) {
  return gatewayFetch<Ledger>(`${F}/reports/general-ledger/${encodeURIComponent(accountId)}${qs({ dateFrom, dateTo })}`);
}

export async function listSubsidiaryParties(kind: LedgerKind) {
  const r = await gatewayFetch<{ parties: SubsidiaryParty[] }>(`${F}/reports/subsidiary-parties${qs({ kind })}`);
  return r.parties;
}

export function getSubsidiaryLedger(kind: LedgerKind, partyName: string, dateFrom?: string, dateTo?: string) {
  return gatewayFetch<SubsidiaryLedger>(`${F}/reports/subsidiary-ledger${qs({ kind, partyName, dateFrom, dateTo })}`);
}

// ── Reimbursements ───────────────────────────────────────────

export function listMyClaims(status?: string) {
  return gatewayFetch<ClaimsResponse>(`${F}/my/reimbursements${qs({ status })}`);
}

export function getClaim(id: string) {
  return gatewayFetch<Claim>(`${F}/reimbursements/${encodeURIComponent(id)}`);
}

export function listApprovalInbox() {
  return gatewayFetch<ClaimsResponse>(`${F}/reimbursement-inbox`);
}

export function listClaims(q: Q) {
  return gatewayFetch<ClaimsResponse>(`${F}/reimbursements${qs(q)}`);
}

export async function listPaymentBatches(limit = 50) {
  const r = await gatewayFetch<{ batches: ReimbursementBatch[] }>(`${F}/reimbursement-batches${qs({ limit })}`);
  return r.batches;
}

export function getMonthlySummary(month?: string, mine = false) {
  return gatewayFetch<MonthlySummary>(`${F}/reimbursements:summary${qs({ month, mine: mine || undefined })}`);
}

export async function listPolicies() {
  const r = await gatewayFetch<{ policies: ReimbursementPolicy[] }>(`${F}/reimbursement-policies`);
  return r.policies;
}

// ── Analysis ─────────────────────────────────────────────────

export function getFinancialAnalysis(from?: string, to?: string, compareMode?: string) {
  return gatewayFetch<FinancialAnalysis>(`${F}/analysis${qs({ from, to, compare: true, compareMode })}`);
}

export async function listFinancialInsights(limit = 20) {
  const r = await gatewayFetch<{ insights: FinancialInsight[] }>(`${F}/analysis/insights${qs({ limit })}`);
  return r.insights;
}
