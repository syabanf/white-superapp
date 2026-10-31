"use server";

import { revalidatePath } from "next/cache";
import { z } from "zod";
import { fail, type ActionResult } from "@/lib/action-result";
import { gatewayFetch } from "@/lib/erp/gateway";
import { runGateway, zodFieldErrors } from "@/features/erp/shared/action";
import { sp } from "./strings";
import type {
  Assignment,
  DailyRecap,
  FeedbackCycle,
  FeedbackSummary,
  IdResponse,
  Improvement,
  Indicator,
  KpiTarget,
  LogbookEntry,
  LogbookTemplate,
  MessageResponse,
  MonthlyRecap,
  OccurrenceAction,
  Retro,
  ReviewAction,
  SaveManualResponse,
  ScorecardDetail,
  SnapshotSummary,
} from "./types";

const P = "/v1/performance";
const id = z.string().min(1);
const ymd = z.string().regex(/^\d{4}-\d{2}-\d{2}$/, sp.common.dateInvalid);
const optText = z.string().trim().max(4000).optional();

/** Sends one JSON request and refreshes the performance and self-service pages. */
async function send<T>(path: string, method: string, body?: unknown): Promise<ActionResult<T>> {
  const res = await runGateway(() =>
    gatewayFetch<T>(`${P}${path}`, { method, body: body === undefined ? undefined : JSON.stringify(body) }),
  );
  if (res.ok) {
    revalidatePath("/erp/performance", "layout");
    revalidatePath("/erp/me", "layout");
  }
  return res;
}

function parse<S extends z.ZodType>(schema: S, input: unknown): { data: z.infer<S> } | { error: ActionResult<never> } {
  const r = schema.safeParse(input);
  if (!r.success) return { error: fail(sp.common.checkForm, "VALIDATION", zodFieldErrors(r.error)) };
  return { data: r.data };
}

const enc = encodeURIComponent;

// ── Tasks ─────────────────────────────────────────────────────

const createTaskSchema = z
  .object({
    departmentId: z.string().optional(),
    assigneeEmployeeId: z.string().optional(),
    title: z.string().trim().min(3, sp.tasks.titleMin).max(200),
    description: z.string().trim().max(4000).optional(),
    recurrence: z.enum(["once", "daily", "weekly", "monthly"]),
    weeklyDay: z.number().int().min(1).max(7).optional(),
    monthlyDay: z.number().int().min(1).max(28).optional(),
    dueDate: ymd.optional(),
    priority: z.enum(["low", "normal", "high", "urgent"]),
    kpiIndicatorId: z.string().optional(),
    kpiContribution: z.number().optional(),
    subtasks: z.array(z.string().trim().min(1).max(200)).max(30).default([]),
  })
  .refine((v) => v.recurrence !== "once" || !!v.dueDate, { path: ["dueDate"], message: sp.tasks.dueRequired });

type CreateTaskInput = z.input<typeof createTaskSchema>;

export async function createTaskAction(input: CreateTaskInput) {
  const p = parse(createTaskSchema, input);
  if ("error" in p) return p.error;
  return send<IdResponse>("/tasks", "POST", p.data);
}

const updateTaskSchema = z.object({
  title: z.string().trim().min(3, sp.tasks.titleMin).max(200),
  description: z.string().trim().max(4000),
  priority: z.enum(["low", "normal", "high", "urgent"]),
  assigneeEmployeeId: z.string(),
  kpiIndicatorId: z.string(),
  kpiContribution: z.number().nullable(),
});

type UpdateTaskInput = z.input<typeof updateTaskSchema>;

export async function updateTaskAction(taskId: string, input: UpdateTaskInput) {
  const p = parse(updateTaskSchema, input);
  if ("error" in p) return p.error;
  const d = p.data;
  return send<MessageResponse>(`/tasks/${enc(taskId)}`, "PATCH", {
    title: d.title,
    description: d.description,
    priority: d.priority,
    ...(d.assigneeEmployeeId ? { assigneeEmployeeId: d.assigneeEmployeeId } : { clearAssignee: true }),
    ...(d.kpiIndicatorId ? { kpiIndicatorId: d.kpiIndicatorId } : { clearKpiIndicator: true }),
    ...(d.kpiContribution != null ? { kpiContribution: d.kpiContribution } : { clearKpiContribution: true }),
  });
}

export async function deactivateTaskAction(taskId: string) {
  return send<MessageResponse>(`/tasks/${enc(taskId)}`, "DELETE");
}

const occurrenceSchema = z.object({
  action: z.enum(["start", "progress", "check_subtask", "comment", "done", "approve", "reject", "reopen"]),
  notes: optText,
  note: optText,
  progressPercent: z.number().int().min(0).max(100).optional(),
  subtaskId: z.string().optional(),
  checked: z.boolean().optional(),
});

export async function occurrenceAction(
  occurrenceId: string,
  input: { action: OccurrenceAction } & Omit<z.input<typeof occurrenceSchema>, "action">,
) {
  const p = parse(occurrenceSchema, input);
  if ("error" in p) return p.error;
  return send<{ message: string; progress?: number; complete: boolean }>(
    `/tasks/occurrences/${enc(occurrenceId)}:act`,
    "POST",
    p.data,
  );
}

export async function deleteAttachmentAction(occurrenceId: string, attachmentId: string) {
  return send<MessageResponse>(`/tasks/occurrences/${enc(occurrenceId)}/attachments/${enc(attachmentId)}`, "DELETE");
}

// ── KPI ───────────────────────────────────────────────────────

const indicatorSchema = z.object({
  id: z.string().default(""),
  code: z
    .string()
    .trim()
    .regex(/^[a-z][a-z0-9_]{1,59}$/, sp.kpi.codeInvalid),
  name: z.string().trim().min(2, sp.common.required).max(120),
  description: z.string().trim().max(500).default(""),
  unit: z.string().trim().max(20).default(""),
  direction: z.enum(["higher_better", "lower_better", "boolean"]),
  sourceKind: z.enum(["auto", "manual"]),
  defaultTarget: z.number().optional(),
  isActive: z.boolean(),
});

export type IndicatorInput = z.input<typeof indicatorSchema>;

export async function saveIndicatorAction(input: IndicatorInput) {
  const p = parse(indicatorSchema, input);
  if ("error" in p) return p.error;
  return send<Indicator>("/kpi/indicators", "POST", p.data);
}

export async function deleteIndicatorAction(indicatorId: string) {
  return send<MessageResponse>(`/kpi/indicators/${enc(indicatorId)}`, "DELETE");
}

const weightsSchema = z.array(z.object({ indicatorId: id, weight: z.number().min(0.01).max(100) })).max(60);

export async function saveDepartmentConfigAction(departmentId: string, items: { indicatorId: string; weight: number }[]) {
  const p = parse(weightsSchema, items);
  if ("error" in p) return p.error;
  return send<MessageResponse>(`/kpi/config/${enc(departmentId)}`, "PUT", { items: p.data });
}

const targetSchema = z.object({
  indicatorId: z.string().min(1, sp.kpi.pickIndicator),
  periodYear: z.number().int().min(2000).max(2100).optional(),
  periodMonth: z.number().int().min(1).max(12).optional(),
  departmentId: z.string().optional(),
  employeeId: z.string().optional(),
  target: z.number({ message: sp.kpi.targetInvalid }).min(0, sp.kpi.targetInvalid),
});

export type TargetInput = z.input<typeof targetSchema>;

export async function createTargetAction(input: TargetInput) {
  const p = parse(targetSchema, input);
  if ("error" in p) return p.error;
  return send<KpiTarget>("/kpi/targets", "POST", p.data);
}

export async function updateTargetAction(targetId: string, input: TargetInput) {
  const p = parse(targetSchema, input);
  if ("error" in p) return p.error;
  return send<KpiTarget>(`/kpi/targets/${enc(targetId)}`, "PATCH", p.data);
}

export async function deleteTargetAction(targetId: string) {
  return send<MessageResponse>(`/kpi/targets/${enc(targetId)}`, "DELETE");
}

const periodSchema = z.object({ periodYear: z.number().int().min(2000), periodMonth: z.number().int().min(1).max(12) });

export async function runSnapshotAction(periodYear: number, periodMonth: number) {
  const p = parse(periodSchema, { periodYear, periodMonth });
  if ("error" in p) return p.error;
  return send<SnapshotSummary>("/kpi/snapshots:run", "POST", p.data);
}

export async function setScorecardStatusAction(scorecardId: string, action: "finalize" | "reopen") {
  return send<{ scorecard: ScorecardDetail["scorecard"]; message: string }>(
    `/kpi/scorecards/${enc(scorecardId)}:status`,
    "POST",
    { action },
  );
}

const manualSchema = z.object({
  employeeId: id,
  indicatorId: z.string().default(""),
  periodYear: z.number().int(),
  periodMonth: z.number().int().min(1).max(12),
  value: z.number({ message: sp.kpi.valueInvalid }),
  notes: optText,
});

type ManualInput = z.input<typeof manualSchema>;

/** Supervisor rubric (no indicator, value 1-5) or a manual indicator's actual. */
export async function saveManualAction(input: ManualInput) {
  const p = parse(manualSchema, input);
  if ("error" in p) return p.error;
  const rubric = !p.data.indicatorId;
  if (rubric && (p.data.value < 1 || p.data.value > 5)) return fail(sp.kpi.rubricRange, "VALIDATION");
  return send<SaveManualResponse>(rubric ? "/kpi/rubric" : "/kpi/actuals", "POST", p.data);
}

// ── Reviews ───────────────────────────────────────────────────

const openCycleSchema = z.object({
  periodYear: z.number().int().min(2000).max(2100),
  periodQuarter: z.number().int().min(1).max(4),
  dueDate: ymd.optional(),
});

export async function openReviewCycleAction(input: z.input<typeof openCycleSchema>) {
  const p = parse(openCycleSchema, input);
  if ("error" in p) return p.error;
  return send<{ message: string }>("/review-cycles", "POST", p.data);
}

export async function closeReviewCycleAction(cycleId: string) {
  return send<MessageResponse>(`/review-cycles/${enc(cycleId)}:close`, "POST", {});
}

const reviewActionSchema = z.object({
  action: z.enum(["self_assessment", "rate_item", "reviewer_notes", "manager_notes", "sign", "finalize"]),
  text: optText,
  itemId: z.string().optional(),
  score: z.number().int().min(1).max(5).optional(),
  notes: optText,
  projectScore: z.number().min(0).max(100).optional(),
});

export async function reviewAction(
  reviewId: string,
  input: { action: ReviewAction } & Omit<z.input<typeof reviewActionSchema>, "action">,
) {
  const p = parse(reviewActionSchema, input);
  if ("error" in p) return p.error;
  return send<MessageResponse>(`/reviews/${enc(reviewId)}:act`, "POST", p.data);
}

// ── 360 feedback ──────────────────────────────────────────────

const feedbackCycleSchema = z.object({
  id: z.string().default(""),
  name: z.string().trim().min(3, sp.common.required).max(120),
  periodLabel: z.string().trim().max(60).default(""),
  startDate: ymd.optional(),
  endDate: ymd.optional(),
  status: z.enum(["draft", "active", "closed"]),
  kpiWeight: z.number().min(0).max(100),
  feedbackWeight: z.number().min(0).max(100),
  isAnonymous: z.boolean(),
  allowSelfAssessment: z.boolean(),
  requireManagerReview: z.boolean(),
}).refine((v) => v.kpiWeight + v.feedbackWeight === 100, { path: ["feedbackWeight"], message: sp.feedback.weightSum });

type FeedbackCycleInput = z.input<typeof feedbackCycleSchema>;

export async function saveFeedbackCycleAction(input: FeedbackCycleInput) {
  const p = parse(feedbackCycleSchema, input);
  if ("error" in p) return p.error;
  return send<FeedbackCycle>("/feedback/cycles", "POST", p.data);
}

export async function deleteFeedbackCycleAction(cycleId: string) {
  return send<MessageResponse>(`/feedback/cycles/${enc(cycleId)}`, "DELETE");
}

export async function autoAssignAction(cycleId: string, peers: number) {
  const n = z.number().int().min(0).max(10).safeParse(peers);
  if (!n.success) return fail(sp.feedback.peersRange, "VALIDATION");
  return send<{ count: number; message: string }>(`/feedback/cycles/${enc(cycleId)}/assignments:auto`, "POST", {
    peers: n.data,
  });
}

const assignmentSchema = z.object({
  employeeId: z.string().min(1, sp.feedback.pickEmployee),
  reviewerId: z.string().min(1, sp.feedback.pickReviewer),
  relationshipType: z.enum(["self", "manager", "peer", "subordinate"]),
  dueDate: ymd.optional(),
});

export async function createAssignmentAction(cycleId: string, input: z.input<typeof assignmentSchema>) {
  const p = parse(assignmentSchema, input);
  if ("error" in p) return p.error;
  return send<{ count: number; message: string }>(`/feedback/cycles/${enc(cycleId)}/assignments`, "POST", {
    items: [p.data],
  });
}

export async function deleteAssignmentAction(assignmentId: string) {
  return send<MessageResponse>(`/feedback/assignments/${enc(assignmentId)}`, "DELETE");
}

const answersSchema = z.object({
  answers: z
    .array(z.object({ criteriaId: id, rating: z.number().int().min(1).max(5), comments: optText }))
    .min(1, sp.feedback.answerOne),
  submit: z.boolean(),
});

export async function submitResponsesAction(assignmentId: string, input: z.input<typeof answersSchema>) {
  const p = parse(answersSchema, input);
  if ("error" in p) return p.error;
  return send<Assignment>(`/feedback/assignments/${enc(assignmentId)}/responses`, "POST", p.data);
}

export async function decideAssignmentAction(
  assignmentId: string,
  input: { approve: boolean; managerComments?: string; rejectionReason?: string },
) {
  if (!input.approve && !input.rejectionReason?.trim()) return fail(sp.feedback.reasonRequired, "VALIDATION");
  return send<{ message: string }>(`/feedback/assignments/${enc(assignmentId)}:decide`, "POST", input);
}

export async function generateSummariesAction(cycleId: string) {
  return send<{ generated: number; improvementsFiled: number; message: string }>(
    `/feedback/cycles/${enc(cycleId)}/summaries:generate`,
    "POST",
    {},
  );
}

const summarySchema = z.object({
  strengths: z.array(z.string().trim().min(1).max(300)).max(20),
  weaknesses: z.array(z.string().trim().min(1).max(300)).max(20),
  managerComments: optText,
  isLocked: z.boolean(),
});

export async function updateSummaryAction(summaryId: string, input: z.input<typeof summarySchema>) {
  const p = parse(summarySchema, input);
  if ("error" in p) return p.error;
  return send<FeedbackSummary>(`/feedback/summaries/${enc(summaryId)}`, "PATCH", p.data);
}

// ── Logbook ───────────────────────────────────────────────────

const templateSchema = z.object({
  departmentId: z.string().min(1, sp.logbook.pickDepartment),
  name: z.string().trim().min(3, sp.common.required).max(120),
  description: optText,
  frequency: z.enum(["daily", "weekly", "monthly"]),
  items: z
    .array(
      z.object({
        title: z.string().trim().min(1).max(200),
        weight: z.number().min(0.1).max(100),
        isRequired: z.boolean(),
      }),
    )
    .min(1, sp.logbook.itemsMin),
});

type TemplateInput = z.input<typeof templateSchema>;

export async function createTemplateAction(input: TemplateInput) {
  const p = parse(templateSchema, input);
  if ("error" in p) return p.error;
  return send<LogbookTemplate>("/logbook/templates", "POST", p.data);
}

export async function deleteTemplateAction(templateId: string) {
  return send<{ message: string; archived: boolean }>(`/logbook/templates/${enc(templateId)}`, "DELETE");
}

export async function createEntryAction(templateId: string, entryDate: string) {
  const p = parse(z.object({ templateId: z.string().min(1, sp.logbook.pickTemplate), entryDate: ymd }), {
    templateId,
    entryDate,
  });
  if ("error" in p) return p.error;
  return send<LogbookEntry>("/logbook/entries", "POST", p.data);
}

export async function updateEntryItemAction(itemId: string, input: { isChecked?: boolean; notes?: string }) {
  return send<LogbookEntry>(`/logbook/items/${enc(itemId)}`, "PATCH", input);
}

export async function submitEntryAction(entryId: string, notes?: string) {
  return send<LogbookEntry>(`/logbook/entries/${enc(entryId)}:submit`, "POST", { notes });
}

export async function reviewEntryAction(entryId: string, status: "reviewed" | "rejected", reviewNotes?: string) {
  if (status === "rejected" && !reviewNotes?.trim()) return fail(sp.logbook.rejectNoteRequired, "VALIDATION");
  return send<LogbookEntry>(`/logbook/entries/${enc(entryId)}:review`, "POST", { status, reviewNotes });
}

export async function deleteEntryAction(entryId: string) {
  return send<MessageResponse>(`/logbook/entries/${enc(entryId)}`, "DELETE");
}

// ── Recaps ────────────────────────────────────────────────────

export async function updateMyDailyNotesAction(date: string, notes: string) {
  const p = parse(z.object({ date: ymd, notes: z.string().max(4000) }), { date, notes });
  if ("error" in p) return p.error;
  return send<DailyRecap>("/recaps/daily/me/notes", "PUT", p.data);
}

export async function regenerateMyDailyRecapAction(date: string) {
  const p = parse(z.object({ date: ymd }), { date });
  if ("error" in p) return p.error;
  return send<DailyRecap>("/recaps/daily/me:regenerate", "POST", p.data);
}

export async function regenerateMonthlyRecapAction(input: {
  scope: "employee" | "department";
  subjectId: string;
  periodYear: number;
  periodMonth: number;
}) {
  const p = parse(
    periodSchema.extend({ scope: z.enum(["employee", "department"]), subjectId: z.string().min(1, sp.recaps.pickSubject) }),
    input,
  );
  if ("error" in p) return p.error;
  return send<{ recap: MonthlyRecap; suggestionsFiled: number }>("/recaps/monthly:regenerate", "POST", p.data);
}

// ── Improvements ──────────────────────────────────────────────

const improvementSchema = z.object({
  scope: z.enum(["employee", "department"]),
  employeeId: z.string().optional(),
  departmentId: z.string().optional(),
  title: z.string().trim().min(3, sp.common.required).max(200),
  description: z.string().trim().max(4000).default(""),
  ownerEmployeeId: z.string().optional(),
  dueDate: ymd.optional(),
  status: z.enum(["open", "in_progress", "done", "dropped"]).optional(),
});

export type ImprovementInput = z.input<typeof improvementSchema>;

export async function createImprovementAction(input: ImprovementInput) {
  const p = parse(
    improvementSchema.refine((v) => (v.scope === "employee" ? !!v.employeeId : !!v.departmentId), {
      path: ["subject"],
      message: sp.improvements.subjectRequired,
    }),
    input,
  );
  if ("error" in p) return p.error;
  return send<Improvement>("/improvements", "POST", p.data);
}

export async function updateImprovementAction(improvementId: string, input: Partial<ImprovementInput>) {
  const p = parse(improvementSchema.partial(), input);
  if ("error" in p) return p.error;
  return send<Improvement>(`/improvements/${enc(improvementId)}`, "PATCH", { ...p.data, id: improvementId });
}

export async function deleteImprovementAction(improvementId: string) {
  return send<MessageResponse>(`/improvements/${enc(improvementId)}`, "DELETE");
}

export async function decideSuggestionAction(
  improvementId: string,
  accept: boolean,
  input: { ownerEmployeeId?: string; dueDate?: string } = {},
) {
  return send<Improvement>(`/improvements/${enc(improvementId)}:${accept ? "accept" : "decline"}`, "POST", input);
}

const retroSchema = z.object({
  departmentId: z.string().min(1, sp.logbook.pickDepartment),
  periodYear: z.number().int(),
  periodMonth: z.number().int().min(1).max(12),
  wentWell: z.array(z.string().trim().min(1).max(500)).max(30),
  toImprove: z.array(z.string().trim().min(1).max(500)).max(30),
  actions: z
    .array(
      z.object({
        description: z.string().trim().min(3).max(500),
        ownerEmployeeId: z.string().optional(),
        dueDate: ymd.optional(),
      }),
    )
    .max(30),
});

type RetroInput = z.input<typeof retroSchema>;

export async function saveRetroAction(input: RetroInput) {
  const p = parse(retroSchema, input);
  if ("error" in p) return p.error;
  return send<Retro>("/retros", "POST", p.data);
}

export async function closeRetroAction(retroId: string) {
  return send<{ improvementsFiled: number; message: string }>(`/retros/${enc(retroId)}:close`, "POST", {});
}
