/** Mirrors white.sales.v1 Dashboard* messages (grpc-gateway JSON, camelCase). */

export type DashboardPeriod = {
  from: string;
  to: string;
  compare: string;
  compareFrom: string;
  compareTo: string;
  /** "YYYY-MM" trend window */
  months: string[];
  compareMonths: string[];
};

export type DashboardKpi = { key: string; value: number; previous: number; spark: number[] };
export type DashboardSeries = { key: string; values: number[]; previous: number[] };
export type DashboardSlice = { key: string; label: string; group: string; value: number; count: number; extra: number };

export type DashboardSummary = {
  period: DashboardPeriod;
  kpis: DashboardKpi[];
  trend: DashboardSeries[];
  stages: DashboardSlice[];
};

export type DashboardDeal = {
  id: string;
  title: string;
  accountName: string;
  stageId: string;
  stageName: string;
  ownerUserId: string;
  ownerName: string;
  value: number;
  daysInStage: number;
  thresholdDays: number;
};

export type DashboardPipeline = {
  period: DashboardPeriod;
  pipelineId: string;
  kpis: DashboardKpi[];
  funnel: DashboardSlice[];
  stageLines: DashboardSlice[];
  aging: DashboardSlice[];
  stageDays: DashboardSlice[];
  stuck: DashboardDeal[];
};

export type DashboardSales = {
  period: DashboardPeriod;
  kpis: DashboardKpi[];
  monthly: DashboardSeries[];
  serviceLines: DashboardSlice[];
  industries: DashboardSlice[];
  lostReasons: DashboardSlice[];
  quoteStatuses: DashboardSlice[];
  sources: DashboardSlice[];
};

export type DashboardMember = {
  userId: string;
  name: string;
  openDeals: number;
  pipelineValue: number;
  wonDeals: number;
  wonValue: number;
  lostDeals: number;
  winRate: number;
  activities: number;
  activitiesDone: number;
  targetValue: number;
  attainmentPercent: number;
  previousWonValue: number;
};

export type DashboardTeam = {
  period: DashboardPeriod;
  kpis: DashboardKpi[];
  members: DashboardMember[];
  wonByOwner: DashboardSeries[];
};

export type DashboardLeads = {
  period: DashboardPeriod;
  kpis: DashboardKpi[];
  monthlySources: DashboardSlice[];
  sources: DashboardSlice[];
  scores: DashboardSlice[];
  response: DashboardSeries[];
  channels: DashboardSlice[];
  forms: DashboardSlice[];
};

export type DashboardInvoice = {
  id: string;
  invoiceNumber: string;
  dealId: string;
  dealTitle: string;
  accountName: string;
  amount: number;
  paid: number;
  dueDate: string;
  daysOverdue: number;
};

export type DashboardTerm = {
  dealId: string;
  dealTitle: string;
  accountName: string;
  contractValue: number;
  invoiced: number;
  paid: number;
  terms: number;
  termsInvoiced: number;
};

export type DashboardBilling = {
  period: DashboardPeriod;
  kpis: DashboardKpi[];
  monthly: DashboardSeries[];
  aging: DashboardSlice[];
  overdue: DashboardInvoice[];
  terms: DashboardTerm[];
};

/** The dashboard query every tab shares; maps 1:1 to DashboardRequest. */
export type DashboardQuery = {
  from: string;
  to: string;
  compare: "previous" | "year";
  pipelineId?: string;
  ownerUserId?: string;
  serviceLine?: string;
  source?: string;
};
