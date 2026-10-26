import type { Metadata } from "next";
import Link from "next/link";
import { notFound } from "next/navigation";
import { ChevronLeft } from "lucide-react";
import { PageHeader } from "@/components/dashboard/page-header";
import { requireErp } from "@/features/erp/access";
import { OfflineNotice } from "@/features/erp/offline-notice";
import { load } from "@/features/erp/shared/load";
import { ErpError } from "@/features/erp/shared/states";
import { ActivityList } from "@/features/erp/sales/components/activities";
import { ConversationRows, DealRows, Panel } from "@/features/erp/sales/components/blocks";
import { CustomFieldValues } from "@/features/erp/sales/components/custom-fields";
import { LeadActions } from "@/features/erp/sales/components/lead-actions";
import { ScoreBadge } from "@/features/erp/sales/components/leads-table";
import { Info, SalesBadge } from "@/features/erp/sales/components/ui";
import { nameOf, salesCaps } from "@/features/erp/sales/lib";
import { getLead, listCustomFields } from "@/features/erp/sales/queries";
import { dealFormRefs } from "@/features/erp/sales/refs";
import { LEAD_SOURCE, LEAD_STATUS, ORG_TYPE, TEMPERATURE, lbl, ss } from "@/features/erp/sales/strings";
import { formatDateTime } from "@/lib/format";

const s = ss.leads;
export const metadata: Metadata = { title: s.title };

export default async function LeadDetailPage(props: PageProps<"/erp/sales/leads/[id]">) {
  const { id } = await props.params;
  const { access, offline } = await requireErp("sales.read");
  if (offline) return <OfflineNotice message={offline.message} />;
  const caps = salesCaps(access);
  const [detail, refs, fields] = await Promise.all([load(() => getLead(id)), dealFormRefs(), load(() => listCustomFields("lead"))]);
  if (detail.error?.code === "NOT_FOUND") notFound();
  if (!detail.data) return <ErpError message={detail.error?.message} />;
  const { lead, deals, activities, conversations } = detail.data;
  const names = Object.fromEntries(refs.owners.map((o) => [o.value, o.label]));
  const account = refs.accounts.find((a) => a.value === lead.accountId);
  const attr = lead.attribution;
  const attrRows = attr
    ? ([
        ["utm_source", attr.utmSource],
        ["utm_medium", attr.utmMedium],
        ["utm_campaign", attr.utmCampaign],
        ["landing", attr.landingPage],
        ["referrer", attr.referrer],
      ] as const).filter(([, v]) => v)
    : [];

  return (
    <>
      <Link href="/erp/sales/leads" className="-mb-3 inline-flex w-fit items-center gap-1 text-sm text-muted-foreground hover:text-foreground">
        <ChevronLeft className="size-4" /> {s.title}
      </Link>
      <PageHeader
        eyebrow={lbl(ORG_TYPE, lead.orgType)}
        title={lead.orgName}
        description={`${lead.picName}${lead.picTitle ? ` · ${lead.picTitle}` : ""}${lead.city ? ` · ${lead.city}` : ""}`}
        actions={caps.write ? <LeadActions lead={lead} refs={refs} leadFields={fields.data ?? []} /> : null}
      />
      <div className="flex flex-wrap items-center gap-2">
        <SalesBadge value={lead.status} map={LEAD_STATUS} />
        <SalesBadge value={lead.temperature} map={TEMPERATURE} />
        <span className="flex items-center gap-1.5 text-sm text-muted-foreground">
          {s.score} <ScoreBadge score={lead.score} />
        </span>
      </div>
      <div className="grid grid-cols-1 gap-4 lg:grid-cols-3">
        <div className="flex flex-col gap-4 lg:col-span-2">
          <Panel title={s.deals} flush>
            <DealRows deals={deals} empty={s.noDeals} />
          </Panel>
          <Panel title={s.activities}>
            <ActivityList activities={activities} owners={refs.owners} canWrite={caps.write} leadId={lead.id} />
          </Panel>
          <Panel title={s.chats} flush>
            <ConversationRows conversations={conversations} empty={s.noChats} />
          </Panel>
        </div>
        <div className="flex flex-col gap-4">
          <Panel title={ss.detail}>
            <dl className="grid grid-cols-2 gap-x-4 gap-y-3">
              <Info label={s.picPhone}>
                {lead.picPhone ? (
                  <a href={`https://wa.me/${lead.picPhone.replace(/\D/g, "")}`} target="_blank" rel="noreferrer" className="tabular hover:underline">
                    {lead.picPhone}
                  </a>
                ) : null}
              </Info>
              <Info label={s.picEmail}>{lead.picEmail}</Info>
              <Info label={s.source}>{lbl(LEAD_SOURCE, lead.source)}</Info>
              <Info label={ss.owner}>{nameOf(names, lead.ownerUserId)}</Info>
              <Info label={ss.accounts.name}>
                {account ? (
                  <Link href={`/erp/sales/leads/accounts/${account.value}`} className="hover:underline">
                    {account.label}
                  </Link>
                ) : null}
              </Info>
              <Info label={ss.createdAt}>{formatDateTime(lead.createdAt)}</Info>
              <CustomFieldValues fields={fields.data ?? []} value={lead.custom ?? {}} />
            </dl>
            {lead.notes ? <p className="mt-4 text-sm whitespace-pre-line text-muted-foreground">{lead.notes}</p> : null}
          </Panel>
          <Panel title={s.scoreBreakdown}>
            {lead.scoreBreakdown.length ? (
              <ul className="space-y-1.5 text-sm">
                {lead.scoreBreakdown.map((b) => (
                  <li key={b.ruleId} className="flex justify-between gap-2">
                    <span className="min-w-0 truncate">{b.name}</span>
                    <span className="font-medium tabular">+{b.points}</span>
                  </li>
                ))}
              </ul>
            ) : (
              <p className="text-sm text-muted-foreground">{ss.none}</p>
            )}
          </Panel>
          {attrRows.length ? (
            <Panel title={s.attribution}>
              <dl className="grid gap-2">
                {attrRows.map(([k, v]) => (
                  <Info key={k} label={k}>
                    {v}
                  </Info>
                ))}
              </dl>
            </Panel>
          ) : null}
        </div>
      </div>
    </>
  );
}
