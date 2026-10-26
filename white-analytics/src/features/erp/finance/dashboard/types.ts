/** Finance dashboard tabs (ReportService dashboard RPCs, AnalysisService trend). Money and ratios are decimal strings; "" means not computable. */

export type AnalyticsWindow = {
  from: string;
  to: string;
  compare: "previous" | "year";
  compareFrom: string;
  compareTo: string;
  months: string[];
  compareMonths: string[];
};

export type AnalyticsKpi = { key: string; value: string; previous: string; trend: string[] };

export type AnalyticsSeries = {
  key: string;
  label: string;
  values: string[];
  compareValues: string[];
  total: string;
  code: string;
};

export type AnalyticsAmount = {
  key: string;
  label: string;
  code: string;
  group: string;
  value: string;
  previous: string;
  count: number;
};

export type WaterfallStep = { key: string; label: string; value: string; total: boolean };

export type FinanceSummary = {
  window: AnalyticsWindow;
  kpis: AnalyticsKpi[];
  pnl: AnalyticsSeries[];
  cash: AnalyticsSeries[];
  assets: AnalyticsAmount[];
  liabilities: AnalyticsAmount[];
  equity: AnalyticsAmount[];
  trialBalanceDiff: string;
  postedEntries: number;
  draftEntries: number;
};

export type FinanceProfitability = {
  window: AnalyticsWindow;
  kpis: AnalyticsKpi[];
  waterfall: WaterfallStep[];
  pnl: AnalyticsSeries[];
  revenueAccounts: AnalyticsAmount[];
  revenueClients: AnalyticsAmount[];
  expenseGroups: AnalyticsAmount[];
  expenseAccounts: AnalyticsAmount[];
  accountMonths: AnalyticsSeries[];
};

export type CashAccountSummary = {
  accountId: string;
  code: string;
  name: string;
  opening: string;
  inflow: string;
  outflow: string;
  closing: string;
};

export type FinanceCash = {
  window: AnalyticsWindow;
  kpis: AnalyticsKpi[];
  flows: AnalyticsSeries[];
  balances: AnalyticsSeries[];
  bridge: WaterfallStep[];
  accounts: CashAccountSummary[];
};

export type AgingRow = { kind: "AR" | "AP"; bucket: string; amount: string; count: number };

export type PartyBalance = {
  partyName: string;
  outstanding: string;
  overdue: string;
  invoiceCount: number;
  oldestDays: number;
  billed: string;
};

export type OverdueInvoice = {
  id: string;
  kind: "AR" | "AP";
  invoiceNo: string;
  partyName: string;
  invoiceDate: string;
  dueDate: string;
  total: string;
  outstanding: string;
  daysOverdue: number;
};

export type FinanceArAp = {
  window: AnalyticsWindow;
  kpis: AnalyticsKpi[];
  aging: AgingRow[];
  arAgingTrend: AnalyticsSeries[];
  apAgingTrend: AnalyticsSeries[];
  days: AnalyticsSeries[];
  customers: PartyBalance[];
  vendors: PartyBalance[];
  overdue: OverdueInvoice[];
};

export type ClaimSummary = {
  id: string;
  claimNo: string;
  employeeName: string;
  departmentName: string;
  category: string;
  transactionDate: string;
  amount: string;
  status: string;
  ageDays: number;
};

export type FinanceSpend = {
  window: AnalyticsWindow;
  kpis: AnalyticsKpi[];
  expenseGroups: AnalyticsSeries[];
  claimCategories: AnalyticsSeries[];
  byCategory: AnalyticsAmount[];
  byDepartment: AnalyticsAmount[];
  byStatus: AnalyticsAmount[];
  leadTime: AnalyticsAmount[];
  leadTimeTrend: AnalyticsSeries[];
  pending: ClaimSummary[];
  departments: string[];
};

export type RatioTrendSeries = {
  key: string;
  label: string;
  group: "profitability" | "liquidity" | "leverage" | "efficiency" | "cash";
  unit: string;
  lowerIsBetter: boolean;
  points: { month: string; value: number; unavailable: boolean }[];
  benchmark?: number | null;
};

export type RatioTrend = { months: string[]; series: RatioTrendSeries[] };

export type DashboardFilters = {
  from: string;
  to: string;
  compare: "previous" | "year";
  accountType?: string;
  accountGroup?: string;
  department?: string;
  cashAccountId?: string;
};
