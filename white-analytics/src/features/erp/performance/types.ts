/**
 * Shapes of white.performance.v1 as grpc-gateway emits them: camelCase,
 * proto3 `optional` fields are omitted when unset, int64 come as strings.
 */

export type Department = { id: string; code: string; name: string; headEmployeeId?: string };

export type Person = {
  id: string;
  name: string;
  departmentId?: string;
  departmentName: string;
  positionName: string;
};

export type Me = {
  userId: string;
  name: string;
  employeeId?: string;
  departmentId?: string;
  headOf: string[];
  directReports: string[];
  hrRead: boolean;
  hrWrite: boolean;
  hrApprove: boolean;
};

export type MessageResponse = { message: string };
export type IdResponse = { id: string; message: string };

// ── KPI ───────────────────────────────────────────────────────

export type Direction = "higher_better" | "lower_better" | "boolean";
export type SourceKind = "auto" | "manual";

export type Indicator = {
  id: string;
  code: string;
  name: string;
  description: string;
  unit: string;
  direction: Direction;
  sourceKind: SourceKind;
  defaultTarget?: number;
  isActive: boolean;
  updatedAt?: string;
};

export type DepartmentWeight = {
  departmentId: string;
  indicatorId: string;
  indicatorCode: string;
  weight: number;
  updatedBy: string;
  updatedAt?: string;
};

export type KpiConfig = { departments: Department[]; indicators: Indicator[]; weights: DepartmentWeight[] };

export type KpiTarget = {
  id: string;
  indicatorId: string;
  periodYear?: number;
  periodMonth?: number;
  departmentId?: string;
  employeeId?: string;
  target: number;
};

export type SnapshotSummary = {
  periodYear: number;
  periodMonth: number;
  employees: number;
  snapshotsUpserted: number;
  scorecardsUpserted: number;
  scorecardsSkippedFinal: number;
  message: string;
};

export type BreakdownRow = {
  code: string;
  weight: number;
  effectiveWeight: number;
  attainment?: number;
  contribution: number;
};

export type Scorecard = {
  id: string;
  employeeId: string;
  employeeName: string;
  departmentId?: string;
  departmentName: string;
  periodYear: number;
  periodMonth: number;
  score?: number;
  rawScore?: number;
  usedWeight: number;
  breakdown: BreakdownRow[];
  status: "draft" | "final";
  reviewedBy?: string;
  reviewedAt?: string;
  updatedAt?: string;
};

export type ListScorecardsResponse = { scorecards: Scorecard[]; indicators: Indicator[] };

export type Snapshot = {
  indicatorId: string;
  indicatorCode: string;
  indicatorName: string;
  sourceKind: SourceKind;
  actual?: number;
  target?: number;
  attainment?: number;
  sampleSize: number;
  sourceDetail: string;
};

export type ScorecardDetail = { scorecard: Scorecard; snapshots: Snapshot[]; canRate: boolean };

export type SaveManualResponse = { score?: number; status: string; message: string };

export type Recommendation = { employeeId: string; avgScore: number; periods: number; hasFinal: boolean };

// ── Reviews ───────────────────────────────────────────────────

export type ReviewCycle = {
  id: string;
  name: string;
  periodYear: number;
  periodQuarter: number;
  startDate: string;
  endDate: string;
  dueDate: string;
  status: "open" | "closed";
  createdByName: string;
  totalReviews: number;
  finalReviews: number;
  ratedReviews: number;
};

export type Review = {
  id: string;
  cycleId: string;
  cycleName: string;
  cycleStatus: string;
  cycleDueDate: string;
  employeeId: string;
  employeeName: string;
  departmentId?: string;
  departmentName: string;
  periodLabel: string;
  startDate: string;
  endDate: string;
  workResultScore?: number;
  behavioralScore?: number;
  projectScore?: number;
  grandTotalScore?: number;
  category?: string;
  status: "draft" | "final";
  selfAssessment?: string;
  reviewerId?: string;
  reviewerName?: string;
  reviewerNotes?: string;
  managerId?: string;
  managerNotes?: string;
  employeeSignDate?: string;
  reviewerSignDate?: string;
  ratedItems: number;
  totalItems: number;
  finalizedAt?: string;
};

export type ReviewItem = {
  id: string;
  valueName: string;
  competency: string;
  behavioralStandard: string;
  scoreDescriptions: string[];
  weight: number;
  score?: number;
  weightedScore?: number;
  notes?: string;
  itemOrder: number;
};

export type ReviewDetail = {
  review: Review;
  items: ReviewItem[];
  kpiMonths: Scorecard[];
  canRate: boolean;
  canFinalize: boolean;
  isOwner: boolean;
};

export type ReviewAction =
  | "self_assessment"
  | "rate_item"
  | "reviewer_notes"
  | "manager_notes"
  | "sign"
  | "finalize";

export type RealtimeRow = {
  employeeId: string;
  employeeName: string;
  departmentName: string;
  avgScore?: number;
  months: Scorecard[];
};

export type RealtimeKpi = { employees: RealtimeRow[]; year: number; quarter: number; months: number[] };

// ── 360 feedback ──────────────────────────────────────────────

export type FeedbackCriterion = { id: string; name: string; description: string; displayOrder: number };

export type FeedbackCategory = {
  id: string;
  code: string;
  name: string;
  description: string;
  weight: number;
  displayOrder: number;
  criteria: FeedbackCriterion[];
};

export type FeedbackCycleStatus = "draft" | "active" | "closed";

export type FeedbackCycle = {
  id: string;
  name: string;
  periodLabel: string;
  startDate?: string;
  endDate?: string;
  status: FeedbackCycleStatus;
  kpiWeight?: number;
  feedbackWeight?: number;
  isAnonymous: boolean;
  allowSelfAssessment: boolean;
  requireManagerReview: boolean;
  assignmentsCount: number;
  submittedCount: number;
  summariesCount: number;
  createdAt?: string;
};

export type Relationship = "self" | "manager" | "peer" | "subordinate";
export type AssignmentStatus = "pending" | "in_progress" | "submitted" | "approved" | "rejected";

export type Assignment = {
  id: string;
  cycleId: string;
  cycleName: string;
  cycleStatus: string;
  employeeId: string;
  employeeName: string;
  reviewerId: string;
  reviewerName: string;
  relationshipType: Relationship;
  status: AssignmentStatus;
  dueDate?: string;
  submittedAt?: string;
  reminderCount: number;
  managerComments?: string;
  rejectionReason?: string;
};

export type FeedbackAnswer = { id: string; criteriaId: string; rating: number; comments?: string };

export type AssignmentDetail = {
  assignment: Assignment;
  responses: FeedbackAnswer[];
  categories: FeedbackCategory[];
  canAnswer: boolean;
  canDecide: boolean;
};

export type FeedbackSummary = {
  id: string;
  cycleId: string;
  employeeId: string;
  employeeName: string;
  departmentName: string;
  scores: Record<string, number>;
  overall360Score?: number;
  kpiScore?: number;
  finalScore?: number;
  finalGrade?: string;
  strengths: string[];
  weaknesses: string[];
  managerComments?: string;
  responsesCount: number;
  isLocked: boolean;
  generatedAt?: string;
};

// ── Tasks ─────────────────────────────────────────────────────

export type TaskStatus = "pending" | "in_progress" | "done" | "approved" | "rejected";
export type Recurrence = "once" | "daily" | "weekly" | "monthly";
export type Priority = "low" | "normal" | "high" | "urgent";

export type TaskIndicator = {
  id: string;
  code: string;
  name: string;
  unit: string;
  direction: string;
  departmentIds: string[];
};

export type TaskMeta = {
  me: Me;
  teamScope: "all" | "heads" | "none";
  departments: Department[];
  members: Person[];
  indicators: TaskIndicator[];
  today: string;
  timezone: string;
};

export type TaskItem = {
  occurrenceId: string;
  taskId: string;
  title: string;
  description: string;
  departmentId: string;
  departmentName: string;
  assigneeEmployeeId?: string;
  assigneeName: string;
  createdByName: string;
  occurrenceDate: string;
  status: TaskStatus;
  progressPercent?: number;
  startedAt?: string;
  doneAt?: string;
  doneByName?: string;
  reviewNotes?: string;
  reviewedByName?: string;
  recurrence: Recurrence;
  priority: Priority;
  reviewRequired: boolean;
  kpiIndicatorId?: string;
  kpiIndicatorCode?: string;
  kpiIndicatorName?: string;
  kpiContribution?: number;
  subtaskCount: number;
  commentCount: number;
  attachmentCount: number;
  lastNote?: string;
  lastNoteBy?: string;
  lastNoteAt?: string;
  progress: number;
  overdue: boolean;
  mine: boolean;
  canWork: boolean;
  canReview: boolean;
  canManage: boolean;
  weeklyDay?: number;
  monthlyDay?: number;
  dueDate?: string;
};

export type Subtask = {
  id: string;
  title: string;
  weight: number;
  sortOrder: number;
  isChecked: boolean;
  checkedByName?: string;
  checkedAt?: string;
};

export type TaskUpdate = {
  id: string;
  kind: "progress" | "comment" | "status" | "review" | "attachment" | "created";
  note?: string;
  progressPercent?: number;
  authorName: string;
  detail: string;
  createdAt?: string;
};

export type TaskAttachment = {
  id: string;
  occurrenceId: string;
  name: string;
  mime: string;
  sizeBytes: string;
  uploadedByName: string;
  createdAt?: string;
};

export type TaskDetail = {
  item: TaskItem;
  subtasks: Subtask[];
  updates: TaskUpdate[];
  attachments: TaskAttachment[];
  contribution?: { indicatorId: string; periodYear: number; periodMonth: number; amount?: number };
};

export type OccurrenceAction =
  | "start"
  | "progress"
  | "check_subtask"
  | "comment"
  | "done"
  | "approve"
  | "reject"
  | "reopen";

export type IndicatorTasks = {
  indicatorId: string;
  code: string;
  name: string;
  unit: string;
  direction: string;
  weight?: number;
  total: number;
  approved: number;
  waitingReview: number;
  open: number;
  overdue: number;
  rate?: number;
  expectedTotal: number;
  contributionTotal: number;
  tasks: TaskItem[];
};

export type MonitorSummary = {
  scheduled: number;
  due: number;
  approved: number;
  waitingReview: number;
  inProgress: number;
  notStarted: number;
  rejected: number;
  overdue: number;
  completion?: number;
  avgProgress?: number;
};

export type MonitorRow = {
  occurrenceId: string;
  taskId: string;
  title: string;
  departmentId: string;
  departmentName: string;
  assigneeId?: string;
  assigneeName?: string;
  occurrenceDate: string;
  status: TaskStatus;
  progress: number;
  lastNote?: string;
};

export type Monitor = {
  start: string;
  end: string;
  today: string;
  summary: MonitorSummary;
  departments: {
    departmentId: string;
    departmentName: string;
    summary: MonitorSummary;
    assignees: { assigneeId?: string; assigneeName: string; summary: MonitorSummary }[];
  }[];
  overdue: MonitorRow[];
  waitingReview: MonitorRow[];
  scope: "all" | "own";
};

// ── Logbook ───────────────────────────────────────────────────

export type LogbookMe = { me: Me; departmentName: string; canReview: boolean; isFullAccess: boolean };

export type Frequency = "daily" | "weekly" | "monthly";

export type LogbookTemplate = {
  id: string;
  departmentId: string;
  departmentName: string;
  name: string;
  description?: string;
  frequency: Frequency;
  isActive: boolean;
  items: { id: string; title: string; description?: string; weight: number; isRequired: boolean; sortOrder: number }[];
  createdAt?: string;
};

export type EntryStatus = "draft" | "submitted" | "reviewed" | "rejected";

export type LogbookEntryItem = {
  id: string;
  title: string;
  description?: string;
  weight: number;
  isRequired: boolean;
  sortOrder: number;
  isChecked: boolean;
  checkedAt?: string;
  notes?: string;
};

export type LogbookEntry = {
  id: string;
  templateId: string;
  templateName: string;
  frequency: Frequency;
  departmentId: string;
  departmentName: string;
  employeeId: string;
  employeeName: string;
  entryDate: string;
  title: string;
  status: EntryStatus;
  completionPercentage: number;
  kpiScore: number;
  notes?: string;
  submittedAt?: string;
  reviewedAt?: string;
  reviewNotes?: string;
  items: LogbookEntryItem[];
};

export type ListEntriesResponse = { entries: LogbookEntry[]; count: number; page: number; limit: number };

export type LogbookSummaryRow = {
  departmentId: string;
  departmentName: string;
  totalEntries: number;
  submittedEntries: number;
  reviewedEntries: number;
  rejectedEntries: number;
  avgCompletion: number;
  avgKpiScore: number;
};

// ── Recaps ────────────────────────────────────────────────────

export type RecapTask = {
  occurrenceId: string;
  title: string;
  status: TaskStatus;
  progress: number;
  completed: boolean;
  priority: Priority;
  subtasks: { title: string; checked: boolean }[];
  notes: string[];
};

export type DailyRecap = {
  id: string;
  employeeId: string;
  employeeName: string;
  departmentId?: string;
  departmentName: string;
  recapDate: string;
  tasks: RecapTask[];
  logbook: { entryId: string; title: string; status: EntryStatus; completion: number; kpiScore: number }[];
  notes: string;
  generatedSummary: string;
  summaryModel: string;
  generatedAt?: string;
  saved: boolean;
};

export type MonthlyMetrics = {
  tasksPlanned: number;
  tasksDone: number;
  tasksOnTime: number;
  tasksOverdue: number;
  completionRate?: number;
  onTimeRate?: number;
  kpiScore?: number;
  logbookEntries: number;
  logbookCompliance?: number;
  workingDays: number;
  presentDays: number;
  lateDays: number;
  absentDays: number;
  leaveDays: number;
  overtimeHours: number;
  hasAttendance: boolean;
  members: number;
  improvementsOpen: number;
};

export type MonthlyRecap = {
  id: string;
  scope: "employee" | "department";
  subjectId: string;
  subjectName: string;
  departmentId?: string;
  departmentName: string;
  periodYear: number;
  periodMonth: number;
  metrics: MonthlyMetrics;
  accomplishments: string[];
  narrative: string;
  narrativeModel: string;
  generatedAt?: string;
};

// ── Improvements ──────────────────────────────────────────────

export type ImprovementSource = "review" | "feedback" | "task_reject" | "logbook_reject" | "retro" | "manual" | "ai";
export type ImprovementStatus = "suggested" | "open" | "in_progress" | "done" | "dropped" | "declined";

export type Improvement = {
  id: string;
  scope: "employee" | "department";
  employeeId?: string;
  employeeName: string;
  departmentId?: string;
  departmentName: string;
  source: ImprovementSource;
  sourceRef?: string;
  title: string;
  description: string;
  ownerEmployeeId?: string;
  ownerName: string;
  dueDate?: string;
  status: ImprovementStatus;
  linkedTaskId?: string;
  completedAt?: string;
  createdAt?: string;
  updatedAt?: string;
  overdue: boolean;
};

export type RetroAction = {
  description: string;
  ownerEmployeeId?: string;
  dueDate?: string;
  improvementId?: string;
};

export type Retro = {
  id: string;
  departmentId: string;
  departmentName: string;
  periodYear: number;
  periodMonth: number;
  wentWell: string[];
  toImprove: string[];
  actions: RetroAction[];
  status: "draft" | "closed";
  closedAt?: string;
  updatedAt?: string;
};
