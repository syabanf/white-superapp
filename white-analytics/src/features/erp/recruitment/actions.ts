"use server";

import { revalidatePath } from "next/cache";
import type { z } from "zod";
import { fail, type ActionResult } from "@/lib/action-result";
import { gatewayFetch } from "@/lib/erp/gateway";
import { runGateway, zodFieldErrors } from "@/features/erp/shared/action";
import { toRupiahInt } from "./lib";
import {
  candidateSchema,
  instrumentSchema,
  interviewSchema,
  jobSchema,
  offerSchema,
  questionSchema,
  scorecardSchema,
  screeningSchema,
} from "./schemas";
import { rs } from "./strings";
import type {
  AiAnalysis,
  AiInterviewSession,
  Candidate,
  Interview,
  JobOpening,
  Offer,
  ProctorEvent,
  PsikotesSession,
  Question,
  TestAnswers,
} from "./types";

const R = "/v1/recruitment";

/** Calls the gateway and refreshes every recruitment page on success. */
async function send<T>(path: string, method: string, body?: unknown): Promise<ActionResult<T>> {
  const res = await runGateway(() =>
    gatewayFetch<T>(path, { method, body: body === undefined ? undefined : JSON.stringify(body) }),
  );
  if (res.ok) revalidatePath("/erp/hr/recruitment", "layout");
  return res;
}

function validate<S extends z.ZodType>(
  schema: S,
  input: unknown,
): { data: z.output<S>; error?: never } | { data?: never; error: ActionResult<never> } {
  const parsed = schema.safeParse(input);
  if (parsed.success) return { data: parsed.data };
  return { error: fail(rs.checkForm, "VALIDATION", zodFieldErrors(parsed.error)) };
}

const orUndef = (v: string) => (v === "" ? undefined : v);

// Job openings

export async function saveJobAction(id: string | null, input: z.input<typeof jobSchema>) {
  const v = validate(jobSchema, input);
  if (v.error) return v.error;
  const body = {
    ...v.data,
    departmentId: orUndef(v.data.departmentId),
    hiringManagerUserId: orUndef(v.data.hiringManagerUserId),
    closingDate: orUndef(v.data.closingDate),
  };
  return id ? send<JobOpening>(`${R}/job-openings/${id}`, "PATCH", body) : send<JobOpening>(`${R}/job-openings`, "POST", body);
}

export async function deleteJobAction(id: string) {
  return send(`${R}/job-openings/${id}`, "DELETE");
}

// Candidates

export async function saveCandidateAction(id: string | null, input: z.input<typeof candidateSchema>) {
  const v = validate(candidateSchema, input);
  if (v.error) return v.error;
  const { status, expectedSalary, ...rest } = v.data;
  const salary = toRupiahInt(expectedSalary);
  if (!id)
    return send<Candidate>(`${R}/candidates`, "POST", {
      ...rest,
      status,
      jobOpeningId: orUndef(rest.jobOpeningId),
      expectedSalary: salary,
    });
  return send<Candidate>(`${R}/candidates/${id}`, "PATCH", {
    ...rest,
    expectedSalary: salary,
    clearExpectedSalary: salary === undefined,
  });
}

export async function deleteCandidateAction(id: string) {
  return send(`${R}/candidates/${id}`, "DELETE");
}

export async function moveStageAction(id: string, status: string) {
  return send<{ status: string; message: string }>(`${R}/candidates/${id}/stage`, "POST", { status });
}

export async function addNoteAction(id: string, content: string) {
  if (!content.trim()) return fail(rs.required, "VALIDATION");
  return send(`${R}/candidates/${id}/notes`, "POST", { content: content.trim() });
}

export async function saveScreeningAction(id: string, input: z.input<typeof screeningSchema>) {
  const v = validate(screeningSchema, input);
  if (v.error) return v.error;
  return send(`${R}/candidates/${id}/screening`, "PUT", {
    ...v.data,
    confirmedSalary: toRupiahInt(v.data.confirmedSalary),
  });
}

export async function deleteCvAction(id: string) {
  return send<Candidate>(`${R}/candidates/${id}/cv`, "DELETE");
}

export async function analyzeCandidateAction(id: string) {
  return send<AiAnalysis>(`${R}/candidates/${id}/ai-analysis`, "POST", {});
}

// Psikotes

export async function invitePsikotesAction(candidateId: string, instrumentIds: string[], expiresDays: number) {
  if (!instrumentIds.length) return fail(rs.psikotes.pickInstrument, "VALIDATION");
  return send<PsikotesSession>(`${R}/candidates/${candidateId}/psikotes/sessions`, "POST", { instrumentIds, expiresDays });
}

export async function savePsikotesSummaryAction(candidateId: string, recommendation: string, notes: string) {
  return send(`${R}/candidates/${candidateId}/psikotes/summary`, "PUT", { recommendation, notes });
}

export async function getTestAnswersAction(testId: string) {
  return runGateway(() => gatewayFetch<TestAnswers>(`${R}/psikotes/session-tests/${testId}/answers`));
}

export async function reviewTestAction(testId: string, reviewNotes: string) {
  return send(`${R}/psikotes/session-tests/${testId}/review`, "PUT", { reviewNotes });
}

export async function listProctorEventsAction(sessionType: "psikotes" | "interview", sessionId: string) {
  return runGateway(() =>
    gatewayFetch<{ events: ProctorEvent[] }>(`${R}/proctor-events/${sessionType}/${sessionId}`).then((r) => r.events),
  );
}

export async function saveInstrumentAction(id: string, input: z.input<typeof instrumentSchema>) {
  const v = validate(instrumentSchema, input);
  if (v.error) return v.error;
  const { questionCount, ...rest } = v.data;
  return send(`${R}/psikotes/instruments/${id}`, "PUT", {
    ...rest,
    questionCount: questionCount ?? undefined,
    clearQuestionCount: questionCount === null,
  });
}

export async function listQuestionsAction(instrumentId: string) {
  return runGateway(() =>
    gatewayFetch<{ questions: Question[] }>(`${R}/psikotes/instruments/${instrumentId}/questions`).then((r) => r.questions),
  );
}

export async function saveQuestionAction(instrumentId: string, id: string | null, input: z.input<typeof questionSchema>) {
  const v = validate(questionSchema, input);
  if (v.error) return v.error;
  const body = { ...v.data, instrumentId };
  return id
    ? send<Question>(`${R}/psikotes/questions/${id}`, "PUT", body)
    : send<Question>(`${R}/psikotes/instruments/${instrumentId}/questions`, "POST", body);
}

export async function deleteQuestionAction(id: string) {
  return send(`${R}/psikotes/questions/${id}`, "DELETE");
}

// Interviews

export async function inviteAiInterviewAction(candidateId: string, expiresDays: number, maxQuestions: number) {
  return send<AiInterviewSession>(`${R}/candidates/${candidateId}/ai-interviews`, "POST", { expiresDays, maxQuestions });
}

export async function saveInterviewAction(id: string | null, input: z.input<typeof interviewSchema>) {
  const v = validate(interviewSchema, input);
  if (v.error) return v.error;
  const body = { ...v.data, interviewerEmployeeId: orUndef(v.data.interviewerEmployeeId) };
  return id ? send<Interview>(`${R}/interviews/${id}`, "PATCH", body) : send<Interview>(`${R}/interviews`, "POST", body);
}

export async function cancelInterviewAction(id: string, reason: string) {
  return send<Interview>(`${R}/interviews/${id}:cancel`, "POST", { reason });
}

export async function submitScorecardAction(id: string, input: z.input<typeof scorecardSchema>) {
  const v = validate(scorecardSchema, input);
  if (v.error) return v.error;
  return send<Interview>(`${R}/interviews/${id}/scorecard`, "PUT", v.data);
}

// Offers

export async function createOfferAction(candidateId: string, input: z.input<typeof offerSchema>) {
  const v = validate(offerSchema, input);
  if (v.error) return v.error;
  return send<Offer>(`${R}/candidates/${candidateId}/offers`, "POST", {
    ...v.data,
    startDate: orUndef(v.data.startDate),
  });
}

export async function recordOfferResponseAction(id: string, status: string, note: string) {
  return send<Offer>(`${R}/offers/${id}/response`, "PUT", { status, note });
}
