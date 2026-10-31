/**
 * JSON shapes of white.sales.v1 as grpc-gateway emits them: camelCase keys,
 * money as decimal strings, days as "YYYY-MM-DD", timestamps as RFC 3339 or
 * null, value sets as the Indonesian strings NüHabit used.
 */

export type Ts = string | null;
export type Custom = Record<string, unknown>;

export type Attribution = {
  utmSource: string;
  utmMedium: string;
  utmCampaign: string;
  utmTerm: string;
  utmContent: string;
  landingPage: string;
  referrer: string;
};

export type ScoreItem = { ruleId: string; name: string; points: number; count: number };

export type Lead = {
  id: string;
  orgName: string;
  orgType: string;
  picName: string;
  picTitle: string;
  picPhone: string;
  picEmail: string;
  city: string;
  source: string;
  temperature: string;
  status: string;
  notes: string;
  ownerUserId: string;
  accountId: string;
  contactId: string;
  custom: Custom;
  score: number;
  scoreBreakdown: ScoreItem[];
  attribution: Attribution | null;
  createdAt: Ts;
  updatedAt: Ts;
};

export type ListLeadsResponse = { leads: Lead[]; total: number };
export type LeadDetail = { lead: Lead; deals: Deal[]; activities: Activity[]; conversations: Conversation[] };
export type ImportLeadsResponse = { imported: number; skipped: number; errors: { row: number; message: string }[] };
export type Owner = { id: string; name: string; email: string };

export type Account = {
  id: string;
  name: string;
  accountType: string;
  industry: string;
  address: string;
  city: string;
  phone: string;
  email: string;
  website: string;
  npwp: string;
  notes: string;
  ownerUserId: string;
  custom: Custom;
  createdAt: Ts;
  updatedAt: Ts;
};
export type AccountDetail = { account: Account; contacts: Contact[]; deals: Deal[] };

export type Contact = {
  id: string;
  accountId: string;
  name: string;
  title: string;
  phone: string;
  email: string;
  isPrimary: boolean;
  notes: string;
  custom: Custom;
  createdAt: Ts;
};

export type CustomField = {
  id: string;
  object: string;
  key: string;
  label: string;
  fieldType: string;
  options: string[];
  isRequired: boolean;
  showInList: boolean;
  sortOrder: number;
  isActive: boolean;
};

export type Stage = {
  id: string;
  pipelineId: string;
  code: string;
  name: string;
  sortOrder: number;
  probability: number;
  isWon: boolean;
  isLost: boolean;
  stuckThresholdDays: number;
  isActive: boolean;
};
export type Pipeline = { id: string; code: string; name: string; isDefault: boolean; sortOrder: number; stages: Stage[] };
export type LostReason = { id: string; code: string; name: string };

export type Deal = {
  id: string;
  leadId: string;
  accountId: string;
  title: string;
  serviceLines: string[];
  pipelineId: string;
  stageId: string;
  stageName: string;
  stageCode: string;
  isWon: boolean;
  isLost: boolean;
  probability: number;
  valueEstimate: string;
  valueFinal: string;
  ownerUserId: string;
  lostReasonId: string;
  expectedCloseDate: string;
  notes: string;
  forecastCategory: string;
  enteredStageAt: Ts;
  closedAt: Ts;
  custom: Custom;
  clientName: string;
  picName: string;
  createdAt: Ts;
  updatedAt: Ts;
};
export type ListDealsResponse = { deals: Deal[]; total: number };

export type DealMember = { userId: string; role: string; splitPercent: number };
export type StageHistoryEntry = { stageId: string; stageName: string; enteredAt: Ts; movedBy: string };
export type TermProgress = {
  termId: string;
  label: string;
  dueDate: string;
  percent: number;
  amount: string;
  paid: string;
  status: string;
};

/** Who a quotation or invoice is addressed to, as printed in its info block. */
export type DocumentParty = {
  deal: Deal;
  payments: Payment[];
  pic: string;
  phone: string;
  email: string;
  city: string;
  owner: string;
};

export type DealDetail = {
  deal: Deal;
  members: DealMember[];
  stageHistory: StageHistoryEntry[];
  quotations: Quotation[];
  invoices: Invoice[];
  payments: Payment[];
  termProgress: TermProgress[];
  referenceTotal: string;
  paidTotal: string;
  paymentStatus: string;
  contracts: Contract[];
  onboarding: Onboarding | null;
};

export type Recurrence = { freq: string; interval: number; until: string };
export type Activity = {
  id: string;
  leadId: string;
  dealId: string;
  subjectType: string;
  subjectId: string;
  activityType: string;
  title: string;
  notes: string;
  priority: string;
  status: string;
  dueAt: Ts;
  doneAt: Ts;
  reminderAt: Ts;
  recurrence: Recurrence | null;
  ownerUserId: string;
  parentTaskId: string;
  createdAt: Ts;
};

export type CatalogTier = { tier: string; price: string; description: string };
export type CatalogItem = {
  id: string;
  code: string;
  name: string;
  serviceLine: string;
  unit: string;
  defaultPrice: string;
  description: string;
  isRetainer: boolean;
  isDefaultForLine: boolean;
  isActive: boolean;
  sortOrder: number;
  tiers: CatalogTier[];
};
export type RetainerDiscount = { minMonths: number; discountPercent: number };

export type QuotationItem = {
  id: string;
  itemType: string;
  catalogId: string;
  description: string;
  scope: string;
  unit: string;
  qty: number;
  unitPrice: string;
  lineTotal: string;
};
export type QuotationTerm = { id: string; label: string; percent: number; dueDate: string; amount: string };
export type Quotation = {
  id: string;
  dealId: string;
  dealTitle: string;
  clientName: string;
  quoteNumber: string;
  version: number;
  parentQuotationId: string;
  status: string;
  usePpn: boolean;
  ppnPercent: number;
  subtotal: string;
  discountPercent: number;
  discountNominal: string;
  ppnNominal: string;
  total: string;
  notes: string;
  validUntil: string;
  approvalStatus: string;
  approvalRequestId: string;
  generatedBy: string;
  items: QuotationItem[];
  terms: QuotationTerm[];
  sentAt: Ts;
  createdAt: Ts;
  updatedAt: Ts;
};

export type QuotationInput = {
  usePpn: boolean;
  ppnPercent: number;
  discountPercent: number;
  notes: string;
  validUntil: string;
  items: { itemType: string; catalogId: string; description: string; scope: string; unit: string; qty: number; unitPrice: string }[];
  terms: { label: string; percent: number; dueDate: string }[];
};

export type GenerateQuotationDraftResponse = { quotation: Quotation; unpricedServiceLines: string[]; model: string };

export type ApprovalRule = {
  id: string;
  name: string;
  level: number;
  minDiscountPercent: number;
  approverRole: string;
  approverUserId: string;
  isActive: boolean;
};
export type ApprovalStep = {
  level: number;
  status: string;
  approverRole: string;
  approverUserId: string;
  decidedBy: string;
  decidedAt: Ts;
  comment: string;
};
export type ApprovalRequest = {
  id: string;
  quotationId: string;
  quoteNumber: string;
  dealId: string;
  dealTitle: string;
  clientName: string;
  status: string;
  currentLevel: number;
  discountPercent: number;
  amount: string;
  requestedBy: string;
  createdAt: Ts;
  resolvedAt: Ts;
  steps: ApprovalStep[];
  canDecide: boolean;
};
export type DecideApprovalResponse = { status: string; nextLevel: number; message: string };

export type Invoice = {
  id: string;
  dealId: string;
  dealTitle: string;
  clientName: string;
  quotationId: string;
  termId: string;
  invoiceNumber: string;
  label: string;
  amount: string;
  dueDate: string;
  status: string;
  sentAt: Ts;
  issuedBy: string;
  cancelReason: string;
  paidTotal: string;
  createdAt: Ts;
};
export type Payment = {
  id: string;
  dealId: string;
  invoiceId: string;
  amount: string;
  method: string;
  paidOn: string;
  note: string;
  createdBy: string;
  createdAt: Ts;
};

export type SalesTarget = { userId: string; targetValue: string; targetDeals: number };
export type ListTargetsResponse = { month: string; targets: SalesTarget[] };
export type ForecastRow = {
  userId: string;
  userName: string;
  targetValue: string;
  targetDeals: number;
  wonValue: string;
  wonDeals: number;
  commitValue: string;
  bestCaseValue: string;
  pipelineValue: string;
  weightedValue: string;
  openDeals: number;
  attainmentPercent: number;
  gap: string;
};
export type Forecast = {
  month: string;
  rows: ForecastRow[];
  total: ForecastRow | null;
  allocatedTargetValue: string;
  companyTargetSet: boolean;
};
export type WorkflowCondition = { field: string; op: string; value: unknown };
export type WorkflowAction = {
  type: string;
  title: string;
  activityType: string;
  notes: string;
  dueInDays: number;
  priority: string;
  assignTo: string;
  userId: string;
  strategy: string;
  userIds: string[];
  onlyIfEmpty: boolean;
  field: string;
  value: unknown;
  to: string;
  message: string;
  role: string;
  hours: number;
  days: number;
};
export type WorkflowRule = {
  id: string;
  name: string;
  object: string;
  triggerType: string;
  triggerConfig: Record<string, unknown>;
  conditions: WorkflowCondition[];
  actions: WorkflowAction[];
  runOncePerRecord: boolean;
  isActive: boolean;
  runCount: number;
  lastRunAt: Ts;
};
export type WorkflowRun = {
  id: string;
  subjectType: string;
  subjectId: string;
  status: string;
  actionsResult: unknown[] | null;
  error: string;
  createdAt: Ts;
};
export type ScoringRule = {
  id: string;
  name: string;
  kind: string;
  field: string;
  operator: string;
  value: unknown;
  eventType: string;
  windowDays: number;
  maxCount: number;
  points: number;
  isActive: boolean;
};

export type FormField = { key: string; label: string; type: string; required: boolean; options: string[]; help: string };
export type LeadForm = {
  id: string;
  slug: string;
  name: string;
  description: string;
  fields: FormField[];
  submitLabel: string;
  successMessage: string;
  redirectUrl: string;
  defaultSource: string;
  leadTemperature: string;
  notifyUserIds: string[];
  isActive: boolean;
  submissionCount: number;
};
export type FormSubmission = {
  id: string;
  leadId: string;
  payload: Record<string, unknown>;
  status: string;
  reason: string;
  createdAt: Ts;
};
export type PublicForm = {
  slug: string;
  name: string;
  description: string;
  fields: FormField[];
  submitLabel: string;
  successMessage: string;
  redirectUrl: string;
};
export type SubmitPublicFormResponse = {
  ok: boolean;
  message: string;
  redirectUrl: string;
  errors: { key: string; message: string }[];
};

export type ContractTemplate = {
  id: string;
  name: string;
  type: string;
  body: string;
  isActive: boolean;
  unknownPlaceholders: string[];
};
export type ListContractTemplatesResponse = { templates: ContractTemplate[]; placeholders: Record<string, string> };
export type Contract = {
  id: string;
  contractNumber: string;
  templateId: string;
  quotationId: string;
  dealId: string;
  type: string;
  title: string;
  clientName: string;
  body: string;
  status: string;
  startDate: string;
  endDate: string;
  signPath: string;
  sentAt: Ts;
  signedAt: Ts;
  signerName: string;
  signerIp: string;
  activatedAt: Ts;
  endedAt: Ts;
  cancelledAt: Ts;
  cancelReason: string;
  createdAt: Ts;
};
export type PublicContract = {
  contractNumber: string;
  title: string;
  clientName: string;
  body: string;
  status: string;
  startDate: string;
  endDate: string;
  signedAt: Ts;
  signerName: string;
};

export type DeckOptions = { tone: string; includePricing: boolean; includeCaseStudies: boolean; extraNotes: string };
export type Slide = { kind: string; title: string; subtitle: string; bullets: string[] };
export type PitchDeck = {
  id: string;
  dealId: string;
  leadId: string;
  title: string;
  version: number;
  options: DeckOptions | null;
  slides: Slide[];
  model: string;
  sizeBytes: string;
  createdBy: string;
  createdAt: Ts;
};
export type CaseStudy = {
  id: string;
  title: string;
  clientName: string;
  industry: string;
  serviceLines: string[];
  challenge: string;
  solution: string;
  results: string;
  sortOrder: number;
  isActive: boolean;
};

export type OnboardingItem = {
  id: string;
  key: string;
  kind: string;
  title: string;
  description: string;
  ownerUserId: string;
  dueDate: string;
  status: string;
  doneAt: Ts;
  doneBy: string;
};
export type OnboardingAsset = { id: string; fileName: string; contentType: string; sizeBytes: string; createdAt: Ts };
export type Onboarding = {
  id: string;
  dealId: string;
  clientName: string;
  status: string;
  portalPath: string;
  brief: Record<string, string>;
  briefSubmittedAt: Ts;
  marketingClientId: string;
  progress: number;
  items: OnboardingItem[];
  assets: OnboardingAsset[];
  createdAt: Ts;
  completedAt: Ts;
};
export type PublicOnboarding = {
  clientName: string;
  progress: number;
  briefFields: FormField[];
  brief: Record<string, string>;
  briefSubmittedAt: Ts;
  assets: OnboardingAsset[];
  steps: { title: string; status: string }[];
};

export type Conversation = {
  id: string;
  channel: string;
  externalId: string;
  leadId: string;
  displayName: string;
  phone: string;
  email: string;
  status: string;
  priority: string;
  assignedUserId: string;
  botActive: boolean;
  handoffReason: string;
  handoffAt: Ts;
  isComplaint: boolean;
  unreadCount: number;
  lastMessageAt: Ts;
  lastMessagePreview: string;
  awaitingSince: Ts;
  firstResponseSeconds: number;
  slaResponseBreached: boolean;
  resolvedAt: Ts;
  createdAt: Ts;
};
export type ChatMessage = {
  id: string;
  direction: string;
  sender: string;
  body: string;
  status: string;
  errorReason: string;
  sentByUserId: string;
  createdAt: Ts;
};
export type ConversationNote = { id: string; authorUserId: string; body: string; createdAt: Ts };
export type ConversationInsight = {
  summary: string;
  topic: string;
  sentiment: string;
  isComplaint: boolean;
  keywords: string[];
  model: string;
  analyzedAt: Ts;
  status: string;
};
export type ConversationDetail = {
  conversation: Conversation;
  messages: ChatMessage[];
  notes: ConversationNote[];
  insight: ConversationInsight | null;
};
export type ReplyTemplate = { id: string; title: string; body: string; isActive: boolean };
export type ChatbotSettings = {
  enabled: boolean;
  botName: string;
  persona: string;
  servicesOffered: string;
  pricingHints: string;
  businessHoursStart: number;
  businessHoursEnd: number;
  handoffRules: string;
  greeting: string;
  minConfidence: number;
  updatedAt: Ts;
  whatsappEnabled: boolean;
  handoffUserIds: string[];
};
export type PublicChat = { token: string; botName: string; messages: ChatMessage[]; humanTakeover: boolean };
