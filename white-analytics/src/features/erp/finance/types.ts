/**
 * JSON shapes of white.finance.v1 as grpc-gateway emits them: camelCase,
 * unpopulated fields present, enums as names, money as decimal strings,
 * proto3 `optional` scalars omitted when unset.
 */

export type EntrySide = "ENTRY_SIDE_UNSPECIFIED" | "DEBIT" | "CREDIT";
export type DocumentStatus = "DOCUMENT_STATUS_UNSPECIFIED" | "DRAFT" | "POSTED" | "VOID";
export type EntryType = "ENTRY_TYPE_UNSPECIFIED" | "MANUAL" | "OPENING" | "AUTO";
export type PeriodStatus = "PERIOD_STATUS_UNSPECIFIED" | "OPEN" | "CLOSED";
export type LedgerKind = "AP" | "AR";
export type CashMovementKind = "CASH_MOVEMENT_KIND_UNSPECIFIED" | "CASH_IN" | "CASH_OUT";
export type CashFlowCategory = "" | "OPERATING" | "INVESTING" | "FINANCING" | "NON_CASH";

// ── Accounts ─────────────────────────────────────────────────

export type AccountType = {
  id: string;
  code: string;
  name: string;
  normalBalance: EntrySide;
  sortOrder: number;
  isActive: boolean;
  accountCount: number;
};

export type AccountTypeInput = Omit<AccountType, "id" | "accountCount">;

export type Account = {
  id: string;
  code: string;
  codeDisplay: string;
  name: string;
  parentId?: string | null;
  accountTypeId: string;
  accountTypeCode: string;
  accountTypeName: string;
  normalBalance: EntrySide;
  level: number;
  isPostable: boolean;
  isContra: boolean;
  isCashBank: boolean;
  cashFlowCategory: CashFlowCategory;
  description: string;
  isActive: boolean;
  createdAt: string;
  updatedAt: string;
};

export type AccountInput = {
  code: string;
  name: string;
  parentId?: string | null;
  accountTypeId: string;
  isContra: boolean;
  isCashBank: boolean;
  cashFlowCategory: string;
  description: string;
  isActive: boolean;
};

export type ImportPreviewRow = {
  sourceRow: number;
  code: string;
  name: string;
  parentCode: string;
  accountTypeCode: string;
  action: "create" | "update" | "skip" | "error" | string;
  message: string;
};

export type ImportAccountsResponse = {
  mode: string;
  issues: { row: number; code: string; message: string }[];
  preview: ImportPreviewRow[];
  createCount: number;
  updateCount: number;
  skipCount: number;
  errorCount: number;
  message: string;
};

export type PostableAccount = {
  id: string;
  code: string;
  codeDisplay: string;
  name: string;
  isCashBank: boolean;
  accountTypeCode: string;
};

export type JournalEvent = { eventCode: string; name: string; description: string; module: string };

export type JournalEventsResponse = { events: JournalEvent[]; modules: string[]; amountSources: string[] };

export type JournalMappingLine = {
  id: string;
  entrySide: EntrySide;
  lineRole: string;
  accountId?: string | null;
  accountCode: string;
  accountName: string;
  amountSource: string;
  sortOrder: number;
  isRequired: boolean;
};

export type JournalMapping = {
  id: string;
  eventCode: string;
  name: string;
  description: string;
  module: string;
  isActive: boolean;
  isReady: boolean;
  lines: JournalMappingLine[];
  updatedAt: string;
};

export type JournalMappingLineInput = {
  entrySide: EntrySide;
  lineRole: string;
  accountId?: string | null;
  amountSource: string;
  sortOrder?: number;
  isRequired?: boolean;
};

export type JournalMappingInput = {
  eventCode: string;
  name: string;
  description: string;
  module: string;
  isActive: boolean;
  lines: JournalMappingLineInput[];
};

// ── Fiscal ───────────────────────────────────────────────────

export type FiscalPeriod = {
  id: string;
  fiscalYearId: string;
  periodNo: number;
  name: string;
  startDate: string;
  endDate: string;
  status: PeriodStatus;
  fiscalYearCode: string;
  fiscalYearName: string;
  fiscalYearIsActive: boolean;
  postedCount: number;
  draftCount: number;
};

export type FiscalYear = {
  id: string;
  code: string;
  name: string;
  startDate: string;
  endDate: string;
  isActive: boolean;
  periods: FiscalPeriod[];
  openPeriodsCount: number;
  createdAt: string;
  updatedAt: string;
};

export type FiscalYearInput = {
  code: string;
  name: string;
  startDate: string;
  endDate: string;
  isActive: boolean;
  periods: { periodNo: number; name: string; startDate: string; endDate: string; status: PeriodStatus }[];
};

export type DraftEntryRef = { id: string; entryNo: string; entryDate: string; description: string };

export type PeriodClosePreview = {
  period: FiscalPeriod;
  postedCount: number;
  draftCount: number;
  draftEntries: DraftEntryRef[];
  canClose: boolean;
  blockers: string[];
};

export type FiscalCoverage = {
  date: string;
  ready: boolean;
  period?: FiscalPeriod | null;
  suggestion?: { period: FiscalPeriod; canOpen: boolean; closePrevious: FiscalPeriod[]; message: string } | null;
};

export type BeginningBalanceLine = {
  accountId: string;
  accountCode: string;
  accountName: string;
  accountTypeCode: string;
  normalBalance: EntrySide;
  isContra: boolean;
  suggestedAmount: string;
  amount: string;
  entrySide: EntrySide;
  source: "PRIOR_BS" | "RETAINED_EARNINGS" | "MANUAL" | string;
};

export type BeginningBalance = {
  fiscalYearId: string;
  fiscalYearCode: string;
  fiscalYearName: string;
  startDate: string;
  periodId: string;
  periodName: string;
  priorFiscalYear?: FiscalYear | null;
  priorFullyClosed: boolean;
  retainedEarningsAccount?: PostableAccount | null;
  lines: BeginningBalanceLine[];
  totalDebit: string;
  totalCredit: string;
  existingEntryId: string;
  existingStatus: DocumentStatus;
  canEdit: boolean;
  isFirstYear: boolean;
  message: string;
};

// ── Journals ─────────────────────────────────────────────────

export type JournalLine = {
  id: string;
  accountId: string;
  accountCode: string;
  accountName: string;
  entrySide: EntrySide;
  amount: string;
  memo: string;
  sortOrder: number;
};

export type JournalEntry = {
  id: string;
  entryNo: string;
  entryDate: string;
  description: string;
  fiscalPeriodId: string;
  fiscalPeriodName: string;
  fiscalYearCode: string;
  entryType: EntryType;
  status: DocumentStatus;
  isRecon: boolean;
  postedAt?: string | null;
  postedBy: string;
  sourceModule: string;
  sourceEventCode: string;
  sourceDocumentType: string;
  sourceDocumentId: string;
  lines: JournalLine[];
  totalDebit: string;
  totalCredit: string;
  canEdit: boolean;
  createdAt: string;
  updatedAt: string;
};

export type JournalLineInput = { accountId: string; entrySide: "DEBIT" | "CREDIT"; amount: string; memo: string };

export type JournalEntryInput = {
  entryDate: string;
  description: string;
  isRecon: boolean;
  lines: JournalLineInput[];
  post: boolean;
};

export type PostingLog = {
  id: string;
  eventCode: string;
  documentType: string;
  documentId: string;
  documentLabel: string;
  status: "posted" | "draft" | "skipped" | string;
  reason: string;
  entryId: string;
  attempts: number;
  createdAt: string;
  updatedAt: string;
};

// ── Cash & bank ──────────────────────────────────────────────

export type CashBankAccount = {
  id: string;
  code: string;
  codeDisplay: string;
  name: string;
  normalBalance: EntrySide;
  balance: string;
  movementCount: number;
};

export type LedgerLine = {
  lineId: string;
  entryId: string;
  entryNo: string;
  entryDate: string;
  entryType: EntryType;
  description: string;
  memo: string;
  entrySide: EntrySide;
  debit: string;
  credit: string;
  runningBalance: string;
};

export type Ledger = {
  account: PostableAccount | null;
  normalBalance: EntrySide;
  dateFrom: string;
  dateTo: string;
  openingBalance: string;
  closingBalance: string;
  totalDebit: string;
  totalCredit: string;
  lines: LedgerLine[];
};

export type CashMovement = {
  id: string;
  entryNo: string;
  entryDate: string;
  description: string;
  kind: CashMovementKind;
  amount: string;
  cashAccount: PostableAccount | null;
  offsetAccount: PostableAccount | null;
  status: DocumentStatus;
  createdAt: string;
};

export type CashTransfer = {
  id: string;
  entryNo: string;
  entryDate: string;
  description: string;
  amount: string;
  fromAccount: PostableAccount | null;
  toAccount: PostableAccount | null;
  status: DocumentStatus;
  createdAt: string;
};

// ── AP / AR ──────────────────────────────────────────────────

export type PaymentStatus = "paid" | "partial" | "overdue" | "unpaid" | string;

export type Invoice = {
  id: string;
  kind: LedgerKind;
  invoiceNo: string;
  invoiceDate: string;
  dueDate: string;
  partyName: string;
  reference: string;
  accountId: string;
  accountName: string;
  currency: string;
  subtotal: string;
  taxAmount: string;
  totalAmount: string;
  status: DocumentStatus;
  description: string;
  salesInvoiceId: string;
  clientId: string;
  dealId: string;
  allocatedAmount: string;
  outstandingAmount: string;
  paymentStatus: PaymentStatus;
  journalEntryId: string;
  postedAt?: string | null;
  createdAt: string;
};

export type InvoiceInput = {
  invoiceDate: string;
  dueDate: string;
  partyName: string;
  reference: string;
  accountId: string;
  subtotal: string;
  taxAmount: string;
  description: string;
  clientId: string;
};

export type AgingBucket = { bucket: string; invoiceCount: number; amount: string };
export type Aging = { asOf: string; buckets: AgingBucket[]; invoices: Invoice[]; totalOutstanding: string };

export type Allocation = { invoiceId: string; invoiceNo: string; amount: string };

export type Payment = {
  id: string;
  kind: LedgerKind;
  paymentNo: string;
  paymentDate: string;
  amount: string;
  method: string;
  cashAccountId: string;
  cashAccountName: string;
  referenceNumber: string;
  notes: string;
  status: DocumentStatus;
  partyName: string;
  allocations: Allocation[];
  voidReason: string;
  voidedAt?: string | null;
  journalEntryId: string;
  createdAt: string;
};

export type PaymentInput = {
  paymentDate: string;
  method: string;
  cashAccountId: string;
  referenceNumber: string;
  notes: string;
  allocations: { invoiceId: string; amount: string }[];
};

export type PaymentResult = { payment: Payment; invoices: Invoice[]; note: string };
export type InvoiceResult = { invoice: Invoice; note: string };

// ── Reports ──────────────────────────────────────────────────

export type AccountBalance = {
  id: string;
  code: string;
  codeDisplay: string;
  name: string;
  accountTypeCode: string;
  normalBalance: EntrySide;
  isContra: boolean;
  isCashBank: boolean;
  cashFlowCategory: string;
  level: number;
  debit: string;
  credit: string;
  balance: string;
};

export type TrialBalance = { asOf: string; rows: AccountBalance[]; totalDebit: string; totalCredit: string };

export type BalanceSheet = {
  asOf: string;
  assets: AccountBalance[];
  liabilities: AccountBalance[];
  equity: AccountBalance[];
  currentYearEarnings: string;
  totalAssets: string;
  totalLiabilities: string;
  totalEquity: string;
  totalLiabilitiesAndEquity: string;
};

export type IncomeStatement = {
  dateFrom: string;
  dateTo: string;
  revenue: AccountBalance[];
  cogs: AccountBalance[];
  expenses: AccountBalance[];
  otherIncome: AccountBalance[];
  otherExpenses: AccountBalance[];
  totalRevenue: string;
  totalCogs: string;
  grossProfit: string;
  totalExpenses: string;
  operatingIncome: string;
  totalOtherIncome: string;
  totalOtherExpenses: string;
  netIncome: string;
};

export type CashFlowSection = { category: string; rows: AccountBalance[]; total: string };

export type CashFlow = {
  dateFrom: string;
  dateTo: string;
  operating: CashFlowSection | null;
  investing: CashFlowSection | null;
  financing: CashFlowSection | null;
  nonCash: CashFlowSection | null;
  netCashChange: string;
  cashOpening: string;
  cashClosing: string;
};

export type GeneralLedgerAccount = {
  id: string;
  code: string;
  codeDisplay: string;
  name: string;
  accountTypeCode: string;
  balance: string;
};

export type SubsidiaryParty = { partyName: string; outstanding: string; invoiceCount: number };

export type SubsidiaryLine = {
  lineId: string;
  entryDate: string;
  docType: "invoice" | "payment" | "receipt" | string;
  docNo: string;
  description: string;
  debit: string;
  credit: string;
  sourceId: string;
  runningBalance: string;
};

export type SubsidiaryLedger = {
  kind: LedgerKind;
  partyName: string;
  dateFrom: string;
  dateTo: string;
  openingBalance: string;
  totalDebit: string;
  totalCredit: string;
  closingBalance: string;
  lines: SubsidiaryLine[];
};

export type NameValue = { name: string; value: string };
export type MonthTrend = { month: string; revenue: string; expense: string; net: string };

export type Dashboard = {
  asOf: string;
  dateFrom: string;
  dateTo: string;
  kpis: {
    totalAssets: string;
    totalLiabilities: string;
    totalEquity: string;
    netIncome: string;
    cashBalance: string;
    cashOpening: string;
    cashChange: string;
    trialBalanceDiff: string;
    postedEntries: number;
    draftEntries: number;
    receivableOutstanding: string;
    payableOutstanding: string;
    pendingReimbursements: number;
  } | null;
  balanceComposition: NameValue[];
  pnlBreakdown: NameValue[];
  cashFlowBreakdown: NameValue[];
  monthlyTrend: MonthTrend[];
  topExpenseAccounts: NameValue[];
};

// ── Reimbursements ───────────────────────────────────────────

export type ClaimCategory = "TRANSPORT" | "MAKAN" | "AKOMODASI" | "OPERASIONAL_PROYEK" | "LAINNYA";
export type ClaimStatus =
  | "CLAIM_DRAFT"
  | "SUBMITTED"
  | "APPROVED_MANAGER"
  | "APPROVED_FINANCE"
  | "PAID"
  | "REJECTED"
  | "CANCELLED";

export type ClaimReceipt = {
  id: string;
  filename: string;
  contentType: string;
  sizeBytes: string;
  createdAt: string;
};

export type ClaimHistory = { action: string; actorId: string; actorName: string; note: string; createdAt: string };

export type Claim = {
  id: string;
  claimNo: string;
  employeeId: string;
  employeeUserId: string;
  employeeName: string;
  departmentName: string;
  managerEmployeeId: string;
  category: ClaimCategory | "CLAIM_CATEGORY_UNSPECIFIED";
  transactionDate: string;
  amount: string;
  description: string;
  clientId: string;
  project: string;
  status: ClaimStatus | "CLAIM_STATUS_UNSPECIFIED";
  overLimit: boolean;
  limitNote: string;
  rejectReason: string;
  rejectedStage: string;
  receipts: ClaimReceipt[];
  history: ClaimHistory[];
  submittedAt?: string | null;
  paidAt?: string | null;
  paymentBatchId: string;
  createdAt: string;
  updatedAt: string;
  canEdit: boolean;
  canApprove: boolean;
};

export type ClaimInput = {
  category: ClaimCategory;
  transactionDate: string;
  amount: string;
  description: string;
  clientId: string;
  project: string;
};

export type ClaimsResponse = { claims: Claim[]; total: number; totalAmount: string };

export type ReimbursementBatch = {
  id: string;
  batchNo: string;
  paidAt: string;
  cashAccountId: string;
  cashAccountName: string;
  totalAmount: string;
  claimCount: number;
  journalEntryId: string;
  journalStatus: string;
  journalReason: string;
  notes: string;
  createdAt: string;
};

export type MonthlySummaryRow = {
  employeeId: string;
  employeeName: string;
  departmentName: string;
  category: ClaimCategory;
  claimCount: number;
  submittedAmount: string;
  approvedAmount: string;
  paidAmount: string;
};

export type MonthlySummary = {
  month: string;
  rows: MonthlySummaryRow[];
  totalSubmitted: string;
  totalApproved: string;
  totalPaid: string;
};

export type ReimbursementPolicy = {
  category: ClaimCategory;
  label: string;
  monthlyLimit: string;
  perClaimLimit: string;
  expenseAccountId: string;
  expenseAccountName: string;
  isActive: boolean;
  autoApprove: boolean;
};

export type PolicyInput = {
  category: ClaimCategory;
  monthlyLimit: string;
  perClaimLimit: string;
  expenseAccountId: string;
  isActive: boolean;
  autoApprove: boolean;
};

// ── Analysis ─────────────────────────────────────────────────

export type ApiDelta = {
  current: number;
  previous: number;
  abs: number;
  pct?: number | null;
  direction: "DIRECTION_UNSPECIFIED" | "UP" | "DOWN" | "FLAT";
};

export type Metric = {
  key: string;
  label: string;
  value: number;
  unavailable: boolean;
  delta?: ApiDelta | null;
  lowerIsBetter: boolean;
  unit: "pct" | "ratio" | "days" | "months" | "idr" | string;
};

export type ExpenseItem = {
  accountId: string;
  accountCode: string;
  accountName: string;
  amount: string;
  previousAmount: string;
  sharePct: number;
  delta?: ApiDelta | null;
};

export type FinancialAnalysis = {
  current: { from: string; to: string } | null;
  previous?: { from: string; to: string } | null;
  profitability: Metric[];
  liquidity: Metric[];
  leverage: Metric[];
  efficiency: Metric[];
  cash: Metric[];
  expenseBreakdown: ExpenseItem[];
  topMovers: ExpenseItem[];
  revenue: string;
  netIncome: string;
  cashBalance: string;
};

export type FinancialInsight = {
  id: string;
  dateFrom: string;
  dateTo: string;
  language: string;
  model: string;
  content: string;
  createdAt: string;
  createdBy: string;
};

export type SaveBeginningBalanceResult = { entryId: string; status: DocumentStatus };
