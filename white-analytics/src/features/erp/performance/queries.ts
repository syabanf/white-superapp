import "server-only";
import { gatewayFetch } from "@/lib/erp/gateway";
import { qs } from "@/features/erp/shared/load";
import type {
  Assignment,
  DailyRecap,
  FeedbackCycle,
  FeedbackSummary,
  Improvement,
  Indicator,
  IndicatorTasks,
  KpiConfig,
  KpiTarget,
  ListEntriesResponse,
  ListScorecardsResponse,
  LogbookMe,
  LogbookSummaryRow,
  LogbookTemplate,
  Monitor,
  MonthlyRecap,
  RealtimeKpi,
  Recommendation,
  Retro,
  Review,
  ReviewCycle,
  TaskItem,
  TaskMeta,
} from "./types";
import type { DashQuery } from "./dashboard/types";

const P = "/v1/performance";
const get = <T>(path: string) => gatewayFetch<T>(`${P}${path}`);

// Tasks
export const getTaskMeta = () => get<TaskMeta>("/tasks/meta");

export async function listTaskItems(f: {
  scope: "mine" | "team";
  view?: string;
  month?: string;
  departmentId?: string;
  assigneeId?: string;
  status?: string;
  indicatorId?: string;
  q?: string;
}) {
  const r = await get<{ items: TaskItem[] }>(`/tasks/items${qs(f)}`);
  return r.items;
}

export const getMonitor = (month: string, departmentId?: string) =>
  get<Monitor>(`/tasks/monitor${qs({ month, departmentId })}`);

export async function getKpiTasks(departmentId: string, month: string) {
  const r = await get<{ indicators: IndicatorTasks[] }>(`/tasks/kpi${qs({ departmentId, month })}`);
  return r.indicators;
}

// KPI
export async function listIndicators(includeInactive = true) {
  const r = await get<{ indicators: Indicator[] }>(`/kpi/indicators${qs({ includeInactive })}`);
  return r.indicators;
}

export const getKpiConfig = () => get<KpiConfig>("/kpi/config");

export async function listTargets() {
  const r = await get<{ targets: KpiTarget[] }>("/kpi/targets");
  return r.targets;
}

export const listScorecards = (f: {
  periodYear?: number;
  periodMonth?: number;
  employeeId?: string;
  team?: boolean;
  history?: number;
}) => get<ListScorecardsResponse>(`/kpi/scorecards${qs(f)}`);

export async function getRecommendations(employeeIds: string[]) {
  if (employeeIds.length === 0) return [];
  const r = await get<{ items: Recommendation[] }>(`/kpi/recommendations${qs({ employeeIds })}`);
  return r.items;
}

// Reviews
export const listReviewCycles = () => get<{ cycles: ReviewCycle[]; canManage: boolean }>("/review-cycles");

export const listReviews = (cycleId: string) =>
  get<{ reviews: Review[]; canReview: boolean }>(`/review-cycles/${encodeURIComponent(cycleId)}/reviews`);

export const getRealtimeKpi = (year: number, quarter: number) =>
  get<RealtimeKpi>(`/kpi/realtime${qs({ year, quarter })}`);

// 360 feedback
export async function listFeedbackCycles(status?: string) {
  const r = await get<{ cycles: FeedbackCycle[] }>(`/feedback/cycles${qs({ status })}`);
  return r.cycles;
}

export async function listAssignments(f: { cycleId?: string; status?: string }) {
  const r = await get<{ assignments: Assignment[] }>(`/feedback/assignments${qs(f)}`);
  return r.assignments;
}

export async function listMyAssignments() {
  const r = await get<{ assignments: Assignment[] }>("/feedback/my-assignments");
  return r.assignments;
}

export async function listSummaries(cycleId: string) {
  const r = await get<{ summaries: FeedbackSummary[] }>(`/feedback/summaries${qs({ cycleId })}`);
  return r.summaries;
}

// Logbook
export const getLogbookMe = () => get<LogbookMe>("/logbook/me");

export async function listLogbookTemplates(departmentId?: string) {
  const r = await get<{ templates: LogbookTemplate[] }>(`/logbook/templates${qs({ departmentId })}`);
  return r.templates;
}

export const listLogbookEntries = (f: {
  departmentId?: string;
  status?: string;
  date?: string;
  from?: string;
  to?: string;
  mine?: boolean;
  page?: number;
  limit?: number;
}) => get<ListEntriesResponse>(`/logbook/entries${qs(f)}`);

export async function getLogbookSummary(f: { departmentId?: string; from?: string; to?: string }) {
  const r = await get<{ departments: LogbookSummaryRow[] }>(`/logbook/summary${qs(f)}`);
  return r.departments;
}

// Recaps
export const getMyDailyRecap = (date?: string) => get<DailyRecap>(`/recaps/daily/me${qs({ date })}`);

export async function listTeamDailyRecaps(date?: string, departmentId?: string) {
  const r = await get<{ recaps: DailyRecap[] }>(`/recaps/daily/team${qs({ date, departmentId })}`);
  return r.recaps;
}

export async function listDailyRecapHistory(employeeId: string, limit = 14) {
  const r = await get<{ recaps: DailyRecap[] }>(`/recaps/daily/history${qs({ employeeId, limit })}`);
  return r.recaps;
}

export async function listMonthlyRecaps(f: {
  periodYear: number;
  periodMonth: number;
  scope?: string;
  departmentId?: string;
  employeeId?: string;
}) {
  const r = await get<{ recaps: MonthlyRecap[] }>(`/recaps/monthly${qs(f)}`);
  return r.recaps;
}

// Improvements
export async function listImprovements(f: {
  scope?: string;
  departmentId?: string;
  status?: string;
  source?: string;
  mine?: boolean;
}) {
  const r = await get<{ improvements: Improvement[] }>(`/improvements${qs(f)}`);
  return r.improvements;
}

export async function listRetros(departmentId?: string) {
  const r = await get<{ retros: Retro[] }>(`/retros${qs({ departmentId })}`);
  return r.retros;
}

// Dashboard
export const getDashboard = <T>(
  section: "context" | "summary" | "kpi" | "tasks" | "logbook" | "reviews" | "improvements",
  q: DashQuery,
) => get<T>(`/dashboard/${section}${qs(q)}`);
