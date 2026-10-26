// JSON shapes of /v1/performance/dashboard/* (grpc-gateway, camelCase).
import type { Department, Person } from "../types";

export type DashScope = "me" | "team" | "all";

export type DashWindow = {
  from: string;
  to: string;
  compareFrom: string;
  compareTo: string;
  compare: string;
  granularity: "day" | "week" | "month";
  scope: DashScope;
  today: string;
};

export type DashContext = {
  window: DashWindow;
  scopes: DashScope[];
  departments: Department[];
  people: Person[];
};

export type DashMetric = { value?: number; previous?: number; spark: number[] };
export type DashCount = { key: string; value: number; previous?: number };

export type DeptMonthScore = {
  departmentId: string;
  departmentName: string;
  period: string;
  score?: number;
  scorecards: number;
};

export type DashSummary = {
  window: DashWindow;
  kpiScore: DashMetric;
  taskCompletion: DashMetric;
  onTimeRate: DashMetric;
  overdueTasks: DashMetric;
  logbookRate: DashMetric;
  openImprovements: DashMetric;
  buckets: string[];
  months: string[];
  departmentTrend: DeptMonthScore[];
  departments: {
    departmentId: string;
    departmentName: string;
    members: number;
    kpiScore?: number;
    kpiScorePrevious?: number;
    taskCompletion?: number;
    onTimeRate?: number;
    overdueTasks: number;
    logbookRate?: number;
    openImprovements: number;
  }[];
};

export type DashKpi = {
  window: DashWindow;
  avgScore: DashMetric;
  finalRate: DashMetric;
  scorecards: DashMetric;
  belowTarget: DashMetric;
  histogram: DashCount[];
  months: string[];
  departmentMonths: DeptMonthScore[];
  indicators: {
    indicatorId: string;
    code: string;
    name: string;
    unit: string;
    direction: string;
    actual?: number;
    target?: number;
    attainment?: number;
    previousAttainment?: number;
    samples: number;
  }[];
  people: {
    employeeId: string;
    name: string;
    departmentId?: string;
    departmentName: string;
    score?: number;
    previous?: number;
    months: number;
    finalMonths: number;
  }[];
  funnel: DashCount[];
};

export type DashTasks = {
  window: DashWindow;
  due: DashMetric;
  completion: DashMetric;
  onTimeRate: DashMetric;
  overdue: DashMetric;
  waitingReview: DashMetric;
  turnaroundHours: DashMetric;
  trend: {
    bucket: string;
    due: number;
    approved: number;
    finished: number;
    onTime: number;
    overdue: number;
    completion?: number;
    onTimeRate?: number;
    previousCompletion?: number;
    previousOnTimeRate?: number;
    turnaroundHours?: number;
  }[];
  departments: {
    departmentId: string;
    departmentName: string;
    due: number;
    pending: number;
    inProgress: number;
    waitingReview: number;
    approved: number;
    rejected: number;
    overdue: number;
    onTimeRate?: number;
  }[];
  aging: DashCount[];
  turnaround: DashCount[];
  people: {
    employeeId: string;
    name: string;
    departmentName: string;
    due: number;
    approved: number;
    onTime: number;
    overdue: number;
    waitingReview: number;
    rejected: number;
    completion?: number;
  }[];
};

export type DashLogbook = {
  window: DashWindow;
  submissionRate: DashMetric;
  avgScore: DashMetric;
  rejectionRate: DashMetric;
  recapCoverage: DashMetric;
  trend: {
    bucket: string;
    expected: number;
    submitted: number;
    reviewed: number;
    rejected: number;
    avgScore?: number;
    recaps: number;
    previousRate?: number;
  }[];
  departments: {
    bucket: string;
    departmentId: string;
    departmentName: string;
    expected: number;
    submitted: number;
    avgScore?: number;
  }[];
  heatmap: { week: string; weekday: number; submitted: number; expected: number }[];
  status: DashCount[];
  people: {
    employeeId: string;
    name: string;
    departmentName: string;
    expected: number;
    submitted: number;
    reviewed: number;
    rejected: number;
    avgScore?: number;
    recaps: number;
  }[];
};

export type CycleProgress = {
  cycleId: string;
  name: string;
  status: string;
  dueDate: string;
  total: number;
  selfAssessed: number;
  rated: number;
  signed: number;
  finalized: number;
  avgScore?: number;
};

export type DashReviews = {
  window: DashWindow;
  cycle: CycleProgress;
  cycles: CycleProgress[];
  ratings: DashCount[];
  reviewScore: DashMetric;
  feedbackCycleId: string;
  feedbackCycleName: string;
  previousFeedbackCycleName: string;
  feedbackProgress: { relationship: string; total: number; submitted: number }[];
  feedbackCategories: { code: string; name: string; score?: number; previous?: number }[];
  feedbackScore: DashMetric;
  feedbackCompletion: DashMetric;
  departments: {
    departmentId: string;
    departmentName: string;
    reviews: number;
    finalized: number;
    avgScore?: number;
    avg360?: number;
  }[];
};

export type DashImprovements = {
  window: DashWindow;
  open: DashMetric;
  created: DashMetric;
  completed: DashMetric;
  overdue: DashMetric;
  aiAcceptance: DashMetric;
  trend: {
    bucket: string;
    manual: number;
    ai: number;
    retro: number;
    automatic: number;
    completed: number;
    aiAccepted: number;
    aiDeclined: number;
  }[];
  status: DashCount[];
  sources: DashCount[];
  overdueItems: {
    id: string;
    title: string;
    scope: string;
    source: string;
    status: string;
    ownerName: string;
    departmentName: string;
    dueDate: string;
    daysOverdue: number;
  }[];
  departments: {
    departmentId: string;
    departmentName: string;
    open: number;
    inProgress: number;
    done: number;
    overdue: number;
    suggested: number;
  }[];
};

export type DashQuery = {
  from: string;
  to: string;
  compare: string;
  departmentId?: string;
  employeeId?: string;
  scope?: string;
};
