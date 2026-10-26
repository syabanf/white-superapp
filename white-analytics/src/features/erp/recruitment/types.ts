/**
 * Recruitment API shapes (white.recruitment.v1 over grpc-gateway): camelCase,
 * int64 as strings, timestamps as ISO strings or null, optional proto fields
 * omitted when unset.
 */

type Ts = string | null;

export type JobOpening = {
  id: string;
  title: string;
  slug: string;
  departmentId?: string;
  department: string;
  location: string;
  employmentType: string;
  workMode: string;
  headcount: number;
  description: string;
  requirements: string;
  benefits: string;
  salaryMin: string;
  salaryMax: string;
  hiringManagerUserId?: string;
  status: string;
  publishedAt: Ts;
  closingDate?: string;
  candidateCount: number;
  hiredCount: number;
  createdAt: Ts;
  updatedAt: Ts;
};

export type Candidate = {
  id: string;
  fullName: string;
  email: string;
  phone: string;
  domicile: string;
  source: string;
  jobOpeningId?: string;
  jobOpeningTitle: string;
  status: string;
  statusLabel: string;
  notes: string;
  lastExperience: string;
  lastEducation: string;
  availability: string;
  expectedSalary?: string;
  hasCv: boolean;
  cvFileName: string;
  hiredAt: Ts;
  createdAt: Ts;
  updatedAt: Ts;
};

export type ListCandidatesResponse = {
  candidates: Candidate[];
  total: number;
  page: number;
  limit: number;
  totalPages: number;
};

export type PipelineColumn = { status: string; label: string; count: number; candidates: Candidate[] };
export type Pipeline = { columns: PipelineColumn[] };

export type CountByKey = { key: string; label: string; count: number };

export type CandidateNote = {
  id: string;
  candidateId: string;
  content: string;
  createdBy?: string;
  createdByName: string;
  createdAt: Ts;
};

export type CandidateActivity = {
  id: string;
  candidateId: string;
  activityType: string;
  description: string;
  createdBy?: string;
  createdByName: string;
  createdAt: Ts;
};

export type Screening = {
  candidateId: string;
  contacted: boolean;
  interested?: boolean;
  availabilityNote: string;
  confirmedSalary?: string;
  willingShift?: boolean;
  willingPlacement?: boolean;
  notes: string;
  recommendation: string;
  updatedByName: string;
  updatedAt: Ts;
};

export type AiAnalysis = {
  candidateId: string;
  summary: string;
  matchScore: number;
  matchReason: string;
  strengths: string[];
  gaps: string[];
  jobContext: string;
  model: string;
  updatedAt: Ts;
};

// Psikotes

export type Instrument = {
  id: string;
  code: string;
  name: string;
  kind: string;
  durationSeconds: number;
  questionCount?: number;
  shuffle: boolean;
  instructions: string;
  isActive: boolean;
  sortOrder: number;
  activeQuestions: number;
};

export type McqOption = { key: string; text: string };
export type PapiStatement = { text: string; scale: string };

export type Question = {
  id: string;
  instrumentId: string;
  body: string;
  options: McqOption[];
  statementA: PapiStatement | null;
  statementB: PapiStatement | null;
  correctKey: string;
  sortOrder: number;
  isActive: boolean;
};

export type ScoreDetail = {
  mcqTotal: number;
  mcqCorrect: number;
  papiScales: Record<string, number>;
  papiDominant: { code: string; label: string; count: number }[];
  papiAnswered: number;
  papiTotal: number;
};

export type SessionTest = {
  id: string;
  sessionId: string;
  instrumentId: string;
  instrumentCode: string;
  instrumentName: string;
  instrumentKind: string;
  status: string;
  score?: number;
  scoreDetail: ScoreDetail | null;
  hasAttachment: boolean;
  reviewNotes: string;
  reviewedByName: string;
  sortOrder: number;
  startedAt: Ts;
  completedAt: Ts;
};

export type PsikotesSession = {
  id: string;
  candidateId: string;
  token: string;
  status: string;
  webcamConsent?: boolean;
  invitedAt: Ts;
  expiresAt: Ts;
  startedAt: Ts;
  completedAt: Ts;
  createdByName: string;
  proctor: { flags: number } | null;
  tests: SessionTest[];
};

export type PsikotesSummary = {
  candidateId: string;
  recommendation: string;
  notes: string;
  updatedByName: string;
  updatedAt: Ts;
};

export type PsikotesPanel = { summary: PsikotesSummary | null; sessions: PsikotesSession[] };

export type AnswerItem = {
  questionId: string;
  body: string;
  options: McqOption[];
  statementA: PapiStatement | null;
  statementB: PapiStatement | null;
  given?: string;
  correctKey: string;
  isCorrect?: boolean;
};

export type TestAnswers = { kind: string; items: AnswerItem[] };

export type ProctorEvent = { id: string; eventType: string; meta: Record<string, string>; createdAt: Ts };

// Interviews

export type InterviewSummary = {
  summary: string;
  relevanceScore: number;
  relevanceConclusion: string;
  relevanceReason: string;
  skills: string[];
  salaryMentioned: boolean;
  salaryValue: string;
  salaryNote: string;
  redFlags: string[];
  followUps: string[];
  limitations: string;
};

export type AiInterviewTurn = {
  id: string;
  turnNo: number;
  topic: string;
  question: string;
  answerTranscript: string;
  answerMode: string;
  askedAt: Ts;
  answeredAt: Ts;
};

export type AiInterviewSession = {
  id: string;
  candidateId: string;
  token: string;
  status: string;
  webcamConsent?: boolean;
  maxQuestions: number;
  aiSummary: InterviewSummary | null;
  summaryModel: string;
  invitedAt: Ts;
  expiresAt: Ts;
  startedAt: Ts;
  completedAt: Ts;
  createdByName: string;
  proctor: { flags: number } | null;
  turns: AiInterviewTurn[];
};

export type ScoreItem = { criterion: string; score: number; note: string };

export type Interview = {
  id: string;
  candidateId: string;
  candidateName: string;
  jobOpeningTitle: string;
  interviewerEmployeeId?: string;
  interviewerUserId?: string;
  interviewerName: string;
  interviewDate: Ts;
  durationMinutes: number;
  type: string;
  mode: string;
  meetingLink: string;
  location: string;
  status: string;
  scorecard: ScoreItem[];
  overallScore?: number;
  recommendation: string;
  notes: string;
  cancelReason: string;
  createdAt: Ts;
  updatedAt: Ts;
};

// Offers

export type Offer = {
  id: string;
  candidateId: string;
  version: number;
  token: string;
  status: string;
  positionTitle: string;
  baseSalary: string;
  benefits: string[];
  startDate?: string;
  notes: string;
  responseNote: string;
  respondedAt: Ts;
  responseSource: string;
  sentAt: Ts;
  expiresAt: Ts;
  createdByName: string;
  createdAt: Ts;
};

export type SalaryReference = {
  expectedSalary?: string;
  interviewExpectation: string;
  positionTitle: string;
  salaryMin: string;
  salaryMax: string;
  confirmedSalary?: string;
};

export type OfferPanel = { salaryReference: SalaryReference | null; offers: Offer[] };

// Public portals

export type PublicJob = {
  slug: string;
  title: string;
  department: string;
  location: string;
  employmentType: string;
  workMode: string;
  headcount: number;
  description: string;
  requirements: string;
  benefits: string;
  publishedAt: Ts;
  closingDate?: string;
};

export type PortalInstrument = { code: string; name: string; kind: string; durationSeconds: number; instructions: string };

export type PortalTest = {
  id: string;
  status: string;
  sortOrder: number;
  startedAt: Ts;
  completedAt: Ts;
  hasAttachment: boolean;
  instrument: PortalInstrument | null;
};

export type PsikotesPortal = {
  status: string;
  candidateName: string;
  positionTitle: string;
  webcamConsent?: boolean;
  expiresAt: Ts;
  startedAt: Ts;
  completedAt: Ts;
  tests: PortalTest[];
};

export type PortalTestRun = {
  test: PortalTest;
  questions: Question[];
  savedAnswers: Record<string, string>;
  endsAt: Ts;
};

export type PortalAction = { status: string; message: string };

export type PortalTurn = {
  id: string;
  turnNo: number;
  question: string;
  answerTranscript: string;
  askedAt: Ts;
  answeredAt: Ts;
};

export type InterviewPortal = {
  status: string;
  candidateName: string;
  positionTitle: string;
  webcamConsent?: boolean;
  maxQuestions: number;
  expiresAt: Ts;
  startedAt: Ts;
  completedAt: Ts;
  turns: PortalTurn[];
  currentTurn: PortalTurn | null;
};

export type InterviewTurnResponse = { done: boolean; turn: PortalTurn | null };

export type OfferPortal = {
  status: string;
  version: number;
  candidateName: string;
  companyName: string;
  positionTitle: string;
  baseSalary: string;
  benefits: string[];
  startDate?: string;
  responseNote: string;
  respondedAt: Ts;
  sentAt: Ts;
  expiresAt: Ts;
};

// ─── Analytics dashboard (GET /v1/recruitment/dashboard) ─────

export type RecruitmentKpis = {
  openPositions: number;
  openHeadcount: number;
  applicants: number;
  inProcess: number;
  offersSent: number;
  offersAccepted: number;
  offersDeclined: number;
  offersExpired: number;
  hires: number;
  avgDaysToHire?: number | null;
  offerAcceptanceRate?: number | null;
};

export type RecruitmentTrendPoint = {
  start: string;
  applications: number;
  hires: number;
  avgDaysToHire?: number | null;
  offersSent: number;
  offersAccepted: number;
  offersDeclined: number;
  offersExpired: number;
  offersNegotiating: number;
  inProcess: number;
  openPositions: number;
  openHeadcount: number;
};

export type FunnelStageStat = { stage: string; label: string; reached: number; current: number; exited: number };

export type StageDurationStat = {
  stage: string;
  label: string;
  count: number;
  avgDays: number;
  minDays: number;
  p25Days: number;
  medianDays: number;
  p75Days: number;
  maxDays: number;
};

export type StuckCandidate = {
  id: string;
  fullName: string;
  jobOpeningId: string;
  jobOpeningTitle: string;
  status: string;
  enteredAt: string;
  daysInStage: number;
};

export type SourceStat = {
  source: string;
  label: string;
  paid: boolean;
  applications: number;
  screened: number;
  interviewed: number;
  offered: number;
  hires: number;
  avgDaysToHire?: number | null;
};

export type OpeningStat = {
  id: string;
  title: string;
  department: string;
  status: string;
  headcount: number;
  openedOn: string;
  windowEnd: string;
  closingDate?: string | null;
  daysOpen: number;
  daysToClose?: number | null;
  applicantsTotal: number;
  applicantsPeriod: number;
  stages: CountByKey[];
  hires: number;
  timeToFillDays?: number | null;
};

export type DashboardOffer = {
  candidateId: string;
  candidateName: string;
  jobOpeningId: string;
  positionTitle: string;
  status: string;
  baseSalary: string;
  salaryMin?: string | null;
  salaryMax?: string | null;
  expectedSalary?: string | null;
  sentAt: string;
  respondedAt?: string | null;
  responseDays?: number | null;
  versions: number;
};

export type RecruitmentDashboard = {
  period: { from: string; to: string };
  comparePeriod: { from: string; to: string };
  compare: "previous" | "year";
  kpis: RecruitmentKpis;
  previousKpis: RecruitmentKpis;
  trend: RecruitmentTrendPoint[];
  previousTrend: RecruitmentTrendPoint[];
  funnel: FunnelStageStat[];
  stageDurations: StageDurationStat[];
  openingStages: { jobOpeningId: string; title: string; status: string; count: number }[];
  stuck: StuckCandidate[];
  stuckAfterDays: number;
  sources: SourceStat[];
  sourceTrend: { start: string; source: string; count: number }[];
  openings: OpeningStat[];
  psikotesStatus: CountByKey[];
  psikotesScores: { instrument: string; bucket: number; count: number; avgScore: number }[];
  aiInterviewStatus: CountByKey[];
  scorecards: { type: string; criterion: string; count: number; avgScore: number }[];
  interviewers: {
    name: string;
    employeeId: string;
    scheduled: number;
    completed: number;
    cancelled: number;
    proceed: number;
    pool: number;
    reject: number;
    avgScore?: number | null;
  }[];
  offers: DashboardOffer[];
};
