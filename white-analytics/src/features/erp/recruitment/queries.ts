import "server-only";
import { gatewayFetch } from "@/lib/erp/gateway";
import { qs } from "@/features/erp/shared/load";
import type {
  AiAnalysis,
  AiInterviewSession,
  Candidate,
  CandidateActivity,
  CandidateNote,
  Instrument,
  Interview,
  JobOpening,
  ListCandidatesResponse,
  OfferPanel,
  Pipeline,
  PsikotesPanel,
  Question,
  RecruitmentDashboard,
  Screening,
} from "./types";

const R = "/v1/recruitment";

export const getDashboard = (p: {
  from: string;
  to: string;
  compare: string;
  jobOpeningId?: string;
  department?: string;
  source?: string;
  stage?: string;
}) => gatewayFetch<RecruitmentDashboard>(`${R}/dashboard${qs(p)}`);

export const listJobOpenings = (p: { status?: string; search?: string } = {}) =>
  gatewayFetch<{ jobOpenings: JobOpening[] }>(`${R}/job-openings${qs(p)}`).then((r) => r.jobOpenings);

export const listCandidates = (p: {
  status?: string;
  jobOpeningId?: string;
  search?: string;
  dateFrom?: string;
  dateTo?: string;
  sort?: string;
  page?: number;
  limit?: number;
  all?: boolean;
}) => gatewayFetch<ListCandidatesResponse>(`${R}/candidates${qs(p)}`);

export const getCandidate = (id: string) => gatewayFetch<Candidate>(`${R}/candidates/${id}`);
export const listNotes = (id: string) =>
  gatewayFetch<{ notes: CandidateNote[] }>(`${R}/candidates/${id}/notes`).then((r) => r.notes);
export const listActivities = (id: string) =>
  gatewayFetch<{ activities: CandidateActivity[] }>(`${R}/candidates/${id}/activities`).then((r) => r.activities);
export const getScreening = (id: string) =>
  gatewayFetch<{ screening: Screening | null }>(`${R}/candidates/${id}/screening`).then((r) => r.screening);
export const getAiAnalysis = (id: string) =>
  gatewayFetch<{ analysis: AiAnalysis | null }>(`${R}/candidates/${id}/ai-analysis`).then((r) => r.analysis);
export const getCandidatePsikotes = (id: string) => gatewayFetch<PsikotesPanel>(`${R}/candidates/${id}/psikotes`);
export const getCandidateAiInterviews = (id: string) =>
  gatewayFetch<{ sessions: AiInterviewSession[] }>(`${R}/candidates/${id}/ai-interviews`).then((r) => r.sessions);
export const getCandidateOffers = (id: string) => gatewayFetch<OfferPanel>(`${R}/candidates/${id}/offers`);

export const getPipeline = (p: { jobOpeningId?: string; search?: string }) => gatewayFetch<Pipeline>(`${R}/pipeline${qs(p)}`);

export const listTalentPool = (p: { search?: string; includeArchived?: boolean; page?: number; limit?: number }) =>
  gatewayFetch<ListCandidatesResponse>(`${R}/talent-pool${qs(p)}`);

export const listInstruments = () =>
  gatewayFetch<{ instruments: Instrument[] }>(`${R}/psikotes/instruments`).then((r) => r.instruments);
export const listQuestions = (instrumentId: string) =>
  gatewayFetch<{ questions: Question[] }>(`${R}/psikotes/instruments/${instrumentId}/questions`).then((r) => r.questions);

export const listInterviews = (p: {
  candidateId?: string;
  status?: string;
  interviewerEmployeeId?: string;
  dateFrom?: string;
  dateTo?: string;
}) => gatewayFetch<{ interviews: Interview[] }>(`${R}/interviews${qs(p)}`).then((r) => r.interviews);

/** Candidates for pickers (name and opening). */
export async function listCandidateOptions() {
  const res = await listCandidates({ all: true });
  return res.candidates.map((c) => ({ value: c.id, label: c.fullName, hint: c.jobOpeningTitle || undefined }));
}
