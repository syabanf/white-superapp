import "server-only";
import { gatewayFetch } from "@/lib/erp/gateway";
import { qs } from "@/features/erp/shared/load";
import type {
  Account,
  AccountDetail,
  Activity,
  ApprovalRequest,
  ApprovalRule,
  CaseStudy,
  CatalogItem,
  ChatbotSettings,
  Contact,
  Contract,
  ConversationDetail,
  Conversation,
  CustomField,
  DealDetail,
  DocumentParty,
  Forecast,
  FormSubmission,
  Invoice,
  LeadDetail,
  LeadForm,
  ListContractTemplatesResponse,
  ListDealsResponse,
  ListLeadsResponse,
  ListTargetsResponse,
  LostReason,
  Onboarding,
  Owner,
  PitchDeck,
  Pipeline,
  Quotation,
  ReplyTemplate,
  RetainerDiscount,
  ScoringRule,
  WorkflowRule,
  WorkflowRun,
} from "./types";

const S = "/v1/sales";
const get = <T>(path: string) => gatewayFetch<T>(`${S}${path}`);

// Leads, accounts, contacts

export const listLeads = (p: {
  q?: string;
  status?: string;
  temperature?: string;
  source?: string;
  ownerUserId?: string;
  dateFrom?: string;
  dateTo?: string;
  scoreMin?: number;
  scoreMax?: number;
  page?: number;
  pageSize?: number;
}) => get<ListLeadsResponse>(`/leads${qs(p)}`);
export const getLead = (id: string) => get<LeadDetail>(`/leads/${id}`);
export const listOwners = async () => (await get<{ owners: Owner[] }>("/owners")).owners;
export const listAccounts = async (p: { q?: string; limit?: number } = {}) =>
  (await get<{ accounts: Account[] }>(`/accounts${qs(p)}`)).accounts;
export const getAccount = (id: string) => get<AccountDetail>(`/accounts/${id}`);
export const listContacts = async (p: { accountId?: string; q?: string; limit?: number } = {}) =>
  (await get<{ contacts: Contact[] }>(`/contacts${qs(p)}`)).contacts;
export const listCustomFields = async (object?: string) =>
  (await get<{ fields: CustomField[] }>(`/custom-fields${qs({ object })}`)).fields;

// Pipeline

export const listPipelines = async () => (await get<{ pipelines: Pipeline[] }>("/pipelines")).pipelines;
export const listLostReasons = async () => (await get<{ reasons: LostReason[] }>("/lost-reasons")).reasons;
export const listDeals = (p: {
  pipelineId?: string;
  stageId?: string;
  ownerUserId?: string;
  leadId?: string;
  status?: string;
  q?: string;
  serviceLine?: string;
  industry?: string;
  source?: string;
  lostReasonId?: string;
  dateFrom?: string;
  dateTo?: string;
  dateField?: "created" | "closed";
  page?: number;
  pageSize?: number;
}) => get<ListDealsResponse>(`/deals${qs(p)}`);
export const getDeal = (id: string) => get<DealDetail>(`/deals/${id}`);

/** Deal plus its lead's PIC contact and owner name, for document headers. */
export async function getDocumentParty(dealId: string): Promise<DocumentParty> {
  const { deal, payments } = await getDeal(dealId);
  const [lead, owners] = await Promise.all([
    deal.leadId ? getLead(deal.leadId).then((r) => r.lead).catch(() => null) : null,
    listOwners().catch(() => []),
  ]);
  return {
    deal,
    payments,
    pic: lead?.picName || deal.picName,
    phone: lead?.picPhone ?? "",
    email: lead?.picEmail ?? "",
    city: lead?.city ?? "",
    owner: owners.find((o) => o.id === deal.ownerUserId)?.name ?? "",
  };
}
export const listActivities = async (p: {
  leadId?: string;
  dealId?: string;
  ownerUserId?: string;
  status?: string;
  openOnly?: boolean;
  mine?: boolean;
  limit?: number;
}) => (await get<{ activities: Activity[] }>(`/activities${qs(p)}`)).activities;

// Quotations

export const listCatalog = async (includeInactive = false) =>
  (await get<{ items: CatalogItem[] }>(`/catalog${qs({ includeInactive: includeInactive || undefined })}`)).items;
export const listRetainerDiscounts = async () =>
  (await get<{ rules: RetainerDiscount[] }>("/retainer-discounts")).rules;
export const listQuotations = async (p: { dealId?: string; status?: string; limit?: number } = {}) =>
  (await get<{ quotations: Quotation[] }>(`/quotations${qs(p)}`)).quotations;
export const getQuotation = (id: string) => get<Quotation>(`/quotations/${id}`);
export const listApprovalRules = async () => (await get<{ rules: ApprovalRule[] }>("/approval-rules")).rules;
export const listApprovals = async (p: { view?: string; status?: string } = {}) =>
  (await get<{ requests: ApprovalRequest[] }>(`/approvals${qs(p)}`)).requests;

// Billing and insight

export const listInvoices = async (
  p: { dealId?: string; status?: string; dateFrom?: string; dateTo?: string; limit?: number } = {},
) =>
  (await get<{ invoices: Invoice[] }>(`/invoices${qs(p)}`)).invoices;
export const getInvoice = (id: string) => get<Invoice>(`/invoices/${id}`);
export const listTargets = (month?: string) => get<ListTargetsResponse>(`/targets${qs({ month })}`);
export const getForecast = (month?: string) => get<Forecast>(`/forecast${qs({ month })}`);

// Automation and forms

export const listWorkflowRules = async () => (await get<{ rules: WorkflowRule[] }>("/workflow-rules")).rules;
export const listWorkflowRuns = async (id: string) =>
  (await get<{ runs: WorkflowRun[] }>(`/workflow-rules/${id}/runs`)).runs;
export const listScoringRules = async () => (await get<{ rules: ScoringRule[] }>("/scoring-rules")).rules;
export const listForms = async () => (await get<{ forms: LeadForm[] }>("/forms")).forms;
export const listFormSubmissions = async (id: string) =>
  (await get<{ submissions: FormSubmission[] }>(`/forms/${id}/submissions`)).submissions;

// Contracts, decks, onboarding

export const listContractTemplates = () => get<ListContractTemplatesResponse>("/contract-templates");
export const listContracts = async (p: { dealId?: string; status?: string; limit?: number } = {}) =>
  (await get<{ contracts: Contract[] }>(`/contracts${qs(p)}`)).contracts;
export const getContract = (id: string) => get<Contract>(`/contracts/${id}`);
export const listPitchDecks = async (p: { dealId?: string; leadId?: string } = {}) =>
  (await get<{ decks: PitchDeck[] }>(`/pitch-decks${qs(p)}`)).decks;
export const getPitchDeck = (id: string) => get<PitchDeck>(`/pitch-decks/${id}`);
export const listCaseStudies = async () => (await get<{ caseStudies: CaseStudy[] }>("/case-studies")).caseStudies;
export const listOnboardings = async (status?: string) =>
  (await get<{ onboardings: Onboarding[] }>(`/onboardings${qs({ status })}`)).onboardings;
export const getOnboarding = (id: string) => get<Onboarding>(`/onboardings/${id}`);

// Chat

export const listConversations = async (p: {
  status?: string;
  channel?: string;
  mine?: boolean;
  q?: string;
  limit?: number;
}) => (await get<{ conversations: Conversation[] }>(`/conversations${qs(p)}`)).conversations;
export const getConversation = (id: string) => get<ConversationDetail>(`/conversations/${id}`);
export const listReplyTemplates = async () => (await get<{ templates: ReplyTemplate[] }>("/reply-templates")).templates;
export const getChatbotSettings = () => get<ChatbotSettings>("/chatbot-settings");
