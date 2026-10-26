import type { Metadata } from "next";
import Link from "next/link";
import { notFound } from "next/navigation";
import { ChevronLeft, Plus } from "lucide-react";
import { PageHeader } from "@/components/dashboard/page-header";
import { Button } from "@/components/ui/button";
import { Progress } from "@/components/ui/progress";
import { requireErp } from "@/features/erp/access";
import { OfflineNotice } from "@/features/erp/offline-notice";
import { load } from "@/features/erp/shared/load";
import { Money } from "@/features/erp/shared/money-display";
import { ErpError } from "@/features/erp/shared/states";
import { ActivityList } from "@/features/erp/sales/components/activities";
import { DealBilling } from "@/features/erp/sales/components/billing";
import { DealStatus, Panel } from "@/features/erp/sales/components/blocks";
import { GenerateContractDialog } from "@/features/erp/sales/components/contracts";
import { CustomFieldValues } from "@/features/erp/sales/components/custom-fields";
import { DealHeaderActions, DealMembers } from "@/features/erp/sales/components/deal-detail";
import { DeckDownload, GeneratePitchDeckDialog } from "@/features/erp/sales/components/decks";
import { ActionButton, Info, SalesBadge } from "@/features/erp/sales/components/ui";
import { GenerateQuotationDialog } from "@/features/erp/sales/components/quotations";
import { startOnboardingAction } from "@/features/erp/sales/actions";
import { nameOf, salesCaps } from "@/features/erp/sales/lib";
import { getDeal, listActivities, listContractTemplates, listLostReasons, listPitchDecks } from "@/features/erp/sales/queries";
import { dealFormRefs } from "@/features/erp/sales/refs";
import {
  CONTRACT_STATUS,
  FORECAST_CATEGORY,
  ONBOARDING_STATUS,
  QUOTATION_STATUS,
  SERVICE_LINE,
  lbl,
  ss,
} from "@/features/erp/sales/strings";
import { formatDate, formatDateTime } from "@/lib/format";

const d = ss.deal;
export const metadata: Metadata = { title: ss.pipeline.title };

export default async function DealPage(props: PageProps<"/erp/sales/pipeline/[id]">) {
  const { id } = await props.params;
  const { access, offline } = await requireErp("sales.read");
  if (offline) return <OfflineNotice message={offline.message} />;
  const caps = salesCaps(access);
  const [detail, refs, reasons, activities, decks, templates] = await Promise.all([
    load(() => getDeal(id)),
    dealFormRefs(),
    load(listLostReasons),
    load(() => listActivities({ dealId: id, limit: 100 })),
    load(() => listPitchDecks({ dealId: id })),
    load(listContractTemplates),
  ]);
  if (detail.error?.code === "NOT_FOUND") notFound();
  if (!detail.data) return <ErpError message={detail.error?.message} />;
  const x = detail.data;
  const deal = x.deal;
  const names = Object.fromEntries(refs.owners.map((o) => [o.value, o.label]));
  const pipeline = refs.pipelines.find((p) => p.id === deal.pipelineId);
  const stages = (pipeline?.stages ?? []).filter((st) => st.isActive).sort((a, b) => a.sortOrder - b.sortOrder);
  const lead = refs.leads.find((l) => l.value === deal.leadId);
  const account = refs.accounts.find((a) => a.value === deal.accountId);
  const accepted = x.quotations.filter((q) => q.status === "diterima");

  return (
    <>
      <Link href="/erp/sales/pipeline" className="-mb-3 inline-flex w-fit items-center gap-1 text-sm text-muted-foreground hover:text-foreground">
        <ChevronLeft className="size-4" /> {ss.pipeline.title}
      </Link>
      <PageHeader
        eyebrow={deal.clientName || ss.pipeline.title}
        title={deal.title}
        description={deal.picName}
        actions={caps.write ? <DealHeaderActions deal={deal} stages={stages} lostReasons={reasons.data ?? []} refs={refs} /> : null}
      />
      <div className="flex flex-wrap items-center gap-2">
        <DealStatus deal={deal} />
        <span className="text-sm text-muted-foreground">
          {ss.pipeline.probability} {deal.probability}% · {lbl(FORECAST_CATEGORY, deal.forecastCategory)}
        </span>
      </div>

      <div className="grid grid-cols-1 gap-4 lg:grid-cols-3">
        <div className="flex flex-col gap-4 lg:col-span-2">
          <Panel
            title={d.quotations}
            flush
            actions={
              caps.write ? (
                <>
                  <Button size="sm" variant="outline" asChild>
                    <Link href={`/erp/sales/quotations/new?deal=${deal.id}`}>
                      <Plus className="size-3.5" /> {ss.quotations.new}
                    </Link>
                  </Button>
                  <GenerateQuotationDialog dealId={deal.id} />
                </>
              ) : null
            }
          >
            {x.quotations.length === 0 ? (
              <p className="px-5 py-4 text-sm text-muted-foreground">{d.noQuotations}</p>
            ) : (
              <ul className="divide-y">
                {x.quotations.map((q) => (
                  <li key={q.id}>
                    <Link href={`/erp/sales/quotations/${q.id}`} className="flex items-center gap-3 px-5 py-3 text-sm hover:bg-accent/50">
                      <span className="min-w-0 flex-1">
                        <span className="font-medium tabular">{q.quoteNumber}</span>
                        <span className="text-xs text-muted-foreground"> v{q.version} · {formatDate(q.createdAt)}</span>
                      </span>
                      <SalesBadge value={q.status} map={QUOTATION_STATUS} />
                      <Money value={q.total} className="w-32 text-right font-medium" />
                    </Link>
                  </li>
                ))}
              </ul>
            )}
          </Panel>

          <Panel title={d.invoices}>
            <DealBilling
              dealId={deal.id}
              caps={caps}
              quotations={x.quotations}
              invoices={x.invoices}
              payments={x.payments}
              termProgress={x.termProgress}
              referenceTotal={x.referenceTotal}
              paidTotal={x.paidTotal}
              paymentStatus={x.paymentStatus}
            />
          </Panel>

          <Panel
            title={d.contracts}
            flush
            actions={
              caps.write && accepted.length ? (
                <GenerateContractDialog quotations={accepted} templates={templates.data?.templates ?? []} />
              ) : null
            }
          >
            {x.contracts.length === 0 ? (
              <p className="px-5 py-4 text-sm text-muted-foreground">{d.noContracts}</p>
            ) : (
              <ul className="divide-y">
                {x.contracts.map((c) => (
                  <li key={c.id}>
                    <Link href={`/erp/sales/contracts/${c.id}`} className="flex items-center gap-3 px-5 py-3 text-sm hover:bg-accent/50">
                      <span className="min-w-0 flex-1">
                        <span className="block font-medium tabular">{c.contractNumber}</span>
                        <span className="block truncate text-xs text-muted-foreground">{c.title}</span>
                      </span>
                      <SalesBadge value={c.status} map={CONTRACT_STATUS} />
                    </Link>
                  </li>
                ))}
              </ul>
            )}
          </Panel>

          <Panel title={ss.leads.activities}>
            <ActivityList activities={activities.data ?? []} owners={refs.owners} canWrite={caps.write} dealId={deal.id} leadId={deal.leadId || undefined} />
          </Panel>
        </div>

        <div className="flex flex-col gap-4">
          <Panel title={d.overview}>
            <dl className="grid grid-cols-2 gap-x-4 gap-y-3">
              <Info label={ss.pipeline.valueEstimate}>
                <Money value={deal.valueEstimate} />
              </Info>
              <Info label={ss.pipeline.valueFinal}>{deal.valueFinal ? <Money value={deal.valueFinal} /> : null}</Info>
              <Info label={ss.pipeline.expectedClose}>{formatDate(deal.expectedCloseDate || null)}</Info>
              <Info label={ss.owner}>{nameOf(names, deal.ownerUserId)}</Info>
              <Info label={ss.pipeline.lead}>
                {lead ? (
                  <Link href={`/erp/sales/leads/${lead.value}`} className="hover:underline">
                    {lead.label}
                  </Link>
                ) : null}
              </Info>
              <Info label={ss.pipeline.account}>
                {account ? (
                  <Link href={`/erp/sales/leads/accounts/${account.value}`} className="hover:underline">
                    {account.label}
                  </Link>
                ) : null}
              </Info>
              <Info label={ss.pipeline.serviceLines}>{deal.serviceLines.map((l) => lbl(SERVICE_LINE, l)).join(", ")}</Info>
              <Info label={ss.createdAt}>{formatDate(deal.createdAt)}</Info>
              <CustomFieldValues fields={refs.customFields} value={deal.custom ?? {}} />
            </dl>
            {deal.notes ? <p className="mt-4 text-sm whitespace-pre-line text-muted-foreground">{deal.notes}</p> : null}
          </Panel>

          <Panel title={d.members}>
            <p className="mb-3 text-xs text-muted-foreground">{d.membersHint}</p>
            <DealMembers dealId={deal.id} members={x.members} owners={refs.owners} canWrite={caps.write} />
          </Panel>

          <Panel title={d.onboarding}>
            {x.onboarding ? (
              <Link href={`/erp/sales/onboarding/${x.onboarding.id}`} className="flex flex-col gap-2 hover:opacity-80">
                <div className="flex items-center justify-between text-sm">
                  <SalesBadge value={x.onboarding.status} map={ONBOARDING_STATUS} />
                  <span className="tabular">{x.onboarding.progress}%</span>
                </div>
                <Progress value={x.onboarding.progress} />
              </Link>
            ) : (
              <div className="flex flex-col gap-3">
                <p className="text-sm text-muted-foreground">{d.noOnboarding}</p>
                {caps.write && deal.isWon ? (
                  <ActionButton size="sm" variant="outline" action={startOnboardingAction.bind(null, deal.id)} success={ss.saved}>
                    {d.startOnboarding}
                  </ActionButton>
                ) : null}
              </div>
            )}
          </Panel>

          <Panel title={d.pitchDeck} actions={caps.write ? <GeneratePitchDeckDialog dealId={deal.id} label={ss.create} variant="outline" /> : null}>
            {decks.data?.length ? (
              <ul className="space-y-2 text-sm">
                {decks.data.map((k) => (
                  <li key={k.id} className="flex items-center justify-between gap-2">
                    <Link href={`/erp/sales/pitch-decks/${k.id}`} className="min-w-0 truncate hover:underline">
                      {ss.decks.version(k.version)} · {formatDateTime(k.createdAt)}
                    </Link>
                    <DeckDownload deck={k} size="xs" />
                  </li>
                ))}
              </ul>
            ) : (
              <p className="text-sm text-muted-foreground">{ss.none}</p>
            )}
          </Panel>

          <Panel title={d.history}>
            <ol className="space-y-2 text-sm">
              {x.stageHistory.map((h, i) => (
                <li key={i} className="flex justify-between gap-2">
                  <span>{h.stageName}</span>
                  <span className="text-xs text-muted-foreground tabular">
                    {formatDateTime(h.enteredAt)}
                    {h.movedBy ? ` · ${nameOf(names, h.movedBy)}` : ""}
                  </span>
                </li>
              ))}
            </ol>
          </Panel>
        </div>
      </div>
    </>
  );
}
