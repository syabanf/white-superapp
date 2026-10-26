"use server";

import { revalidatePath } from "next/cache";
import { z } from "zod";
import { fail, type ActionResult } from "@/lib/action-result";
import { gatewayFetch } from "@/lib/erp/gateway";
import { runGateway, zodFieldErrors } from "@/features/erp/shared/action";
import { ss } from "./strings";
import type {
  Account,
  Activity,
  ApprovalRule,
  CaseStudy,
  CatalogItem,
  ChatbotSettings,
  Contact,
  Contract,
  ContractTemplate,
  Conversation,
  ConversationDetail,
  ConversationInsight,
  ConversationNote,
  CustomField,
  Deal,
  DealMember,
  DecideApprovalResponse,
  GenerateQuotationDraftResponse,
  ImportLeadsResponse,
  Invoice,
  Lead,
  LeadForm,
  ListTargetsResponse,
  Onboarding,
  Payment,
  PitchDeck,
  Quotation,
  QuotationInput,
  ReplyTemplate,
  RetainerDiscount,
  SalesTarget,
  ScoringRule,
  Stage,
  WorkflowRule,
} from "./types";

const S = "/v1/sales";

/** Calls the gateway and refreshes every Sales page on success. */
async function send<T>(path: string, method: string, body?: unknown): Promise<ActionResult<T>> {
  const res = await runGateway(() =>
    gatewayFetch<T>(`${S}${path}`, { method, body: body === undefined ? undefined : JSON.stringify(body) }),
  );
  if (res.ok) revalidatePath("/erp/sales", "layout");
  return res;
}

function validate<S extends z.ZodType>(
  schema: S,
  input: unknown,
): { data: z.output<S>; error?: never } | { data?: never; error: ActionResult<never> } {
  const parsed = schema.safeParse(input);
  if (parsed.success) return { data: parsed.data };
  return { error: fail(ss.required, "VALIDATION", zodFieldErrors(parsed.error)) };
}

const req = (msg = "Wajib diisi") => z.string().trim().min(1, msg);
const opt = z.string().trim().default("");
const money = z.string().regex(/^\d+(\.\d{1,2})?$/, "Nominal tidak valid");
const moneyOpt = z.union([money, z.literal("")]).default("");
const day = z.union([z.string().regex(/^\d{4}-\d{2}-\d{2}$/, "Tanggal tidak valid"), z.literal("")]).default("");
const custom = z.record(z.string(), z.unknown()).default({});

// Leads

const leadSchema = z.object({
  orgName: req(),
  orgType: req(),
  picName: req(),
  picTitle: opt,
  picPhone: z.string().trim().min(8, "Nomor minimal 8 digit"),
  picEmail: z.union([z.email("Email tidak valid"), z.literal("")]).default(""),
  city: opt,
  source: req(),
  temperature: req(),
  status: req(),
  notes: opt,
  ownerUserId: opt,
  accountId: opt,
  custom,
});
export type LeadValues = z.input<typeof leadSchema>;

export async function saveLeadAction(id: string | null, input: LeadValues) {
  const v = validate(leadSchema, input);
  if (v.error) return v.error;
  return id ? send<Lead>(`/leads/${id}`, "PATCH", { ...v.data, id }) : send<Lead>("/leads", "POST", v.data);
}

export async function deleteLeadAction(id: string) {
  return send(`/leads/${id}`, "DELETE");
}

/** File bytes as base64 (proto bytes field); limit 5 MB. */
export async function importLeadsAction(form: FormData) {
  const file = form.get("file");
  if (!(file instanceof File) || file.size === 0) return fail(ss.leads.importMissing, "VALIDATION");
  if (file.size > 5 * 1024 * 1024) return fail(ss.leads.importTooLarge, "VALIDATION");
  const data = Buffer.from(await file.arrayBuffer()).toString("base64");
  return send<ImportLeadsResponse>("/leads:import", "POST", { fileName: file.name, data });
}

// Accounts and contacts

const accountSchema = z.object({
  name: req(),
  accountType: opt,
  industry: opt,
  address: opt,
  city: opt,
  phone: opt,
  email: z.union([z.email("Email tidak valid"), z.literal("")]).default(""),
  website: opt,
  npwp: opt,
  notes: opt,
  ownerUserId: opt,
});
export type AccountValues = z.input<typeof accountSchema>;

export async function saveAccountAction(id: string | null, input: AccountValues) {
  const v = validate(accountSchema, input);
  if (v.error) return v.error;
  return id ? send<Account>(`/accounts/${id}`, "PATCH", { ...v.data, id }) : send<Account>("/accounts", "POST", v.data);
}

export async function deleteAccountAction(id: string) {
  return send(`/accounts/${id}`, "DELETE");
}

const contactSchema = z.object({
  accountId: opt,
  name: req(),
  title: opt,
  phone: opt,
  email: z.union([z.email("Email tidak valid"), z.literal("")]).default(""),
  isPrimary: z.boolean().default(false),
  notes: opt,
});
export type ContactValues = z.input<typeof contactSchema>;

export async function saveContactAction(id: string | null, input: ContactValues) {
  const v = validate(contactSchema, input);
  if (v.error) return v.error;
  return id ? send<Contact>(`/contacts/${id}`, "PATCH", { ...v.data, id }) : send<Contact>("/contacts", "POST", v.data);
}

export async function deleteContactAction(id: string) {
  return send(`/contacts/${id}`, "DELETE");
}

export async function saveCustomFieldAction(input: CustomField) {
  if (!input.key.trim() || !input.label.trim()) return fail(ss.required, "VALIDATION");
  return send<CustomField>("/custom-fields", "POST", input);
}

export async function deleteCustomFieldAction(id: string) {
  return send(`/custom-fields/${id}`, "DELETE");
}

// Deals and pipeline

const dealSchema = z.object({
  leadId: opt,
  accountId: opt,
  title: req(),
  serviceLines: z.array(z.string()).default([]),
  pipelineId: opt,
  stageId: opt,
  valueEstimate: moneyOpt,
  ownerUserId: opt,
  expectedCloseDate: day,
  notes: opt,
  forecastCategory: opt,
  custom,
});
export type DealValues = z.input<typeof dealSchema>;

export async function saveDealAction(id: string | null, input: DealValues) {
  const v = validate(dealSchema, input);
  if (v.error) return v.error;
  return id ? send<Deal>(`/deals/${id}`, "PATCH", { ...v.data, id }) : send<Deal>("/deals", "POST", v.data);
}

export async function deleteDealAction(id: string) {
  return send(`/deals/${id}`, "DELETE");
}

export async function moveDealAction(id: string, stageId: string, extra: { valueFinal?: string; lostReasonId?: string } = {}) {
  return send<Deal>(`/deals/${id}:move`, "POST", { stageId, ...extra });
}

export async function setDealMembersAction(id: string, members: DealMember[]) {
  return send<{ members: DealMember[] }>(`/deals/${id}/members`, "PUT", { members });
}

export async function saveStageAction(input: Stage) {
  if (!input.name.trim()) return fail(ss.required, "VALIDATION");
  return send<Stage>("/stages", "POST", input);
}

// Activities

const activitySchema = z.object({
  leadId: opt,
  dealId: opt,
  activityType: req(),
  title: req(),
  notes: opt,
  priority: z.string().default("normal"),
  status: z.string().default("open"),
  dueAt: z.string().nullable().default(null),
  reminderAt: z.string().nullable().default(null),
  recurrence: z.object({ freq: z.string(), interval: z.number().int().min(1), until: z.string() }).nullable().default(null),
  ownerUserId: opt,
});
export type ActivityValues = z.input<typeof activitySchema>;

export async function saveActivityAction(id: string | null, input: ActivityValues) {
  const v = validate(activitySchema, input);
  if (v.error) return v.error;
  return id
    ? send<Activity>(`/activities/${id}`, "PATCH", { ...v.data, id })
    : send<Activity>("/activities", "POST", v.data);
}

export async function setActivityStatusAction(a: Activity, status: string) {
  const { id, leadId, dealId, activityType, title, notes, priority, dueAt, reminderAt, recurrence, ownerUserId } = a;
  return send<Activity>(`/activities/${id}`, "PATCH", {
    id,
    leadId,
    dealId,
    activityType,
    title,
    notes,
    priority,
    status,
    dueAt,
    reminderAt,
    recurrence,
    ownerUserId,
  });
}

export async function deleteActivityAction(id: string) {
  return send(`/activities/${id}`, "DELETE");
}

// Quotations

export async function createQuotationAction(dealId: string, input: QuotationInput) {
  if (!input.items.length) return fail(ss.quotations.noItems, "VALIDATION");
  return send<Quotation>(`/deals/${dealId}/quotations`, "POST", { ...input, dealId });
}

export async function updateQuotationAction(id: string, input: QuotationInput) {
  if (!input.items.length) return fail(ss.quotations.noItems, "VALIDATION");
  return send<Quotation>(`/quotations/${id}`, "PATCH", { ...input, id });
}

export async function generateQuotationAction(
  dealId: string,
  input: { tier: string; retainerMonths: number; useAi: boolean; usePpn: boolean; startDate: string },
) {
  return send<GenerateQuotationDraftResponse>(`/deals/${dealId}/quotations:generate`, "POST", input);
}

export async function setQuotationStatusAction(id: string, status: string) {
  return send<Quotation>(`/quotations/${id}:status`, "POST", { status });
}

export async function reviseQuotationAction(id: string) {
  return send<Quotation>(`/quotations/${id}:revise`, "POST", {});
}

export async function deleteQuotationAction(id: string) {
  return send(`/quotations/${id}`, "DELETE");
}

export async function decideApprovalAction(id: string, decision: "approve" | "reject", comment: string) {
  return send<DecideApprovalResponse>(`/approvals/${id}:decide`, "POST", { decision, comment });
}

export async function saveApprovalRuleAction(input: ApprovalRule) {
  if (!input.name.trim()) return fail(ss.required, "VALIDATION");
  return send<ApprovalRule>("/approval-rules", "POST", input);
}

export async function deleteApprovalRuleAction(id: string) {
  return send(`/approval-rules/${id}`, "DELETE");
}

export async function saveCatalogItemAction(input: CatalogItem) {
  if (!input.name.trim() || !input.code.trim()) return fail(ss.required, "VALIDATION");
  return send<CatalogItem>("/catalog", "POST", input);
}

export async function saveRetainerDiscountsAction(rules: RetainerDiscount[]) {
  return send<{ rules: RetainerDiscount[] }>("/retainer-discounts", "PUT", { rules });
}

// Billing

export async function submitInvoiceAction(
  dealId: string,
  input: { quotationId: string; termId: string; label: string; amount: string; dueDate: string },
) {
  if (!input.termId && !money.safeParse(input.amount).success) return fail("Nominal tidak valid", "VALIDATION");
  return send<Invoice>(`/deals/${dealId}/invoices`, "POST", input);
}

export async function invoicesFromTermsAction(quotationId: string) {
  return send<{ invoices: Invoice[] }>(`/quotations/${quotationId}:invoices`, "POST", {});
}

export async function issueInvoiceAction(id: string) {
  return send<Invoice>(`/invoices/${id}:issue`, "POST", {});
}

export async function cancelInvoiceAction(id: string, reason: string) {
  return send<Invoice>(`/invoices/${id}:cancel`, "POST", { reason });
}

const paymentSchema = z.object({
  invoiceId: opt,
  amount: money,
  method: req(),
  paidOn: z.string().regex(/^\d{4}-\d{2}-\d{2}$/, "Tanggal tidak valid"),
  note: opt,
});
export type PaymentValues = z.input<typeof paymentSchema>;

export async function recordPaymentAction(dealId: string, input: PaymentValues) {
  const v = validate(paymentSchema, input);
  if (v.error) return v.error;
  return send<Payment>(`/deals/${dealId}/payments`, "POST", v.data);
}

export async function deletePaymentAction(id: string) {
  return send(`/payments/${id}`, "DELETE");
}

// Insight

export async function saveTargetsAction(month: string, targets: SalesTarget[]) {
  return send<ListTargetsResponse>("/targets", "PUT", { month, targets });
}

// Automation

export async function saveWorkflowRuleAction(input: WorkflowRule) {
  if (!input.name.trim()) return fail(ss.required, "VALIDATION");
  return send<WorkflowRule>("/workflow-rules", "POST", input);
}

export async function deleteWorkflowRuleAction(id: string) {
  return send(`/workflow-rules/${id}`, "DELETE");
}

export async function saveScoringRuleAction(input: ScoringRule) {
  if (!input.name.trim()) return fail(ss.required, "VALIDATION");
  return send<ScoringRule>("/scoring-rules", "POST", input);
}

export async function deleteScoringRuleAction(id: string) {
  return send(`/scoring-rules/${id}`, "DELETE");
}

export async function recalculateScoresAction() {
  return send<{ leads: number; changed: number }>("/scoring:recalculate", "POST", {});
}

// Forms

export async function saveFormAction(input: LeadForm) {
  if (!input.name.trim() || !input.slug.trim()) return fail(ss.required, "VALIDATION");
  if (!input.fields.length) return fail(ss.forms.noFields, "VALIDATION");
  return send<LeadForm>("/forms", "POST", input);
}

export async function deleteFormAction(id: string) {
  return send(`/forms/${id}`, "DELETE");
}

// Contracts

export async function generateContractAction(
  quotationId: string,
  input: { templateId: string; startDate: string; endDate: string; title: string },
) {
  if (!input.templateId) return fail(ss.required, "VALIDATION");
  return send<Contract>(`/quotations/${quotationId}/contracts`, "POST", input);
}

export async function updateContractAction(
  id: string,
  input: { title: string; body: string; startDate: string; endDate: string },
) {
  if (!input.title.trim() || !input.body.trim()) return fail(ss.required, "VALIDATION");
  return send<Contract>(`/contracts/${id}`, "PATCH", { ...input, id });
}

export async function sendContractAction(id: string) {
  return send<Contract>(`/contracts/${id}:send`, "POST", {});
}

export async function setContractStatusAction(id: string, status: string, reason = "") {
  return send<Contract>(`/contracts/${id}:status`, "POST", { status, reason });
}

export async function saveContractTemplateAction(input: ContractTemplate) {
  if (!input.name.trim() || !input.body.trim()) return fail(ss.required, "VALIDATION");
  return send<ContractTemplate>("/contract-templates", "POST", input);
}

export async function deleteContractTemplateAction(id: string) {
  return send(`/contract-templates/${id}`, "DELETE");
}

// Pitch decks

export async function generatePitchDeckAction(input: {
  dealId?: string;
  leadId?: string;
  options: { tone: string; includePricing: boolean; includeCaseStudies: boolean; extraNotes: string };
}) {
  if (!input.dealId && !input.leadId) return fail(ss.required, "VALIDATION");
  return send<PitchDeck>("/pitch-decks", "POST", input);
}

export async function saveCaseStudyAction(input: CaseStudy) {
  if (!input.title.trim() || !input.clientName.trim()) return fail(ss.required, "VALIDATION");
  return send<CaseStudy>("/case-studies", "POST", input);
}

export async function deleteCaseStudyAction(id: string) {
  return send(`/case-studies/${id}`, "DELETE");
}

// Onboarding

export async function startOnboardingAction(dealId: string) {
  return send<Onboarding>(`/deals/${dealId}/onboarding`, "POST", {});
}

export async function updateOnboardingItemAction(
  id: string,
  patch: { status?: string; ownerUserId?: string; dueDate?: string },
) {
  return send<Onboarding>(`/onboarding-items/${id}`, "PATCH", patch);
}

export async function createMarketingClientAction(
  id: string,
  input: { name: string; slug: string; websiteUrl: string; industry: string },
) {
  if (!input.name.trim()) return fail(ss.required, "VALIDATION");
  const res = await send<Onboarding>(`/onboardings/${id}:create-client`, "POST", input);
  if (res.ok) revalidatePath("/", "layout");
  return res;
}

// Chat

export async function replyConversationAction(id: string, body: string) {
  if (!body.trim()) return fail(ss.required, "VALIDATION");
  return send<ConversationDetail>(`/conversations/${id}:reply`, "POST", { body });
}

export async function conversationAction(id: string, action: string, extra: { status?: string; value?: boolean } = {}) {
  return send<Conversation>(`/conversations/${id}:action`, "POST", { action, ...extra });
}

export async function addConversationNoteAction(id: string, body: string) {
  if (!body.trim()) return fail(ss.required, "VALIDATION");
  return send<ConversationNote>(`/conversations/${id}/notes`, "POST", { body });
}

export async function analyzeConversationAction(id: string) {
  return send<ConversationInsight>(`/conversations/${id}:analyze`, "POST", {});
}

export async function saveReplyTemplateAction(input: ReplyTemplate) {
  if (!input.title.trim() || !input.body.trim()) return fail(ss.required, "VALIDATION");
  return send<ReplyTemplate>("/reply-templates", "POST", input);
}

export async function deleteReplyTemplateAction(id: string) {
  return send(`/reply-templates/${id}`, "DELETE");
}

export async function saveChatbotSettingsAction(input: ChatbotSettings) {
  return send<ChatbotSettings>("/chatbot-settings", "PUT", input);
}
