import type { Metadata } from "next";
import { notFound } from "next/navigation";
import { DetailLayout } from "@/components/dashboard/layout";
import { PageHeader, Section } from "@/components/dashboard/page-header";
import { requireErp } from "@/features/erp/access";
import { OfflineNotice } from "@/features/erp/offline-notice";
import { load } from "@/features/erp/shared/load";
import { ErpError } from "@/features/erp/shared/states";
import { ContactsTable } from "@/features/erp/sales/components/accounts";
import { DealRows, Panel } from "@/features/erp/sales/components/blocks";
import { AccountActions } from "@/features/erp/sales/components/lead-actions";
import { Info } from "@/features/erp/sales/components/ui";
import { nameOf, salesCaps } from "@/features/erp/sales/lib";
import { getAccount } from "@/features/erp/sales/queries";
import { ownerOptions } from "@/features/erp/sales/refs";
import { ORG_TYPE, lbl, ss } from "@/features/erp/sales/strings";

const a = ss.accounts;
export const metadata: Metadata = { title: ss.leads.tabs.accounts };

export default async function AccountDetailPage(props: PageProps<"/erp/sales/leads/accounts/[id]">) {
  const { id } = await props.params;
  const { access, offline } = await requireErp("sales.read");
  if (offline) return <OfflineNotice message={offline.message} />;
  const caps = salesCaps(access);
  const [detail, owners] = await Promise.all([load(() => getAccount(id)), ownerOptions()]);
  if (detail.error?.code === "NOT_FOUND") notFound();
  if (!detail.data) return <ErpError message={detail.error?.message} />;
  const { account, contacts, deals } = detail.data;
  const names = Object.fromEntries(owners.map((o) => [o.value, o.label]));

  return (
    <>
      <PageHeader
        back={{ href: "/erp/sales/leads?tab=accounts", label: ss.leads.tabs.accounts }}
        eyebrow={lbl(ORG_TYPE, account.accountType)}
        title={account.name}
        description={[account.industry, account.city].filter(Boolean).join(" · ")}
        actions={caps.write ? <AccountActions account={account} owners={owners} /> : null}
      />
      <DetailLayout
        aside={
          <Panel title={ss.detail}>
            <dl className="grid grid-cols-2 gap-x-4 gap-y-3">
              <Info label={a.phone}>{account.phone}</Info>
              <Info label={a.email}>{account.email}</Info>
              <Info label={a.website}>{account.website}</Info>
              <Info label={a.npwp}>{account.npwp}</Info>
              <Info label={ss.owner}>{nameOf(names, account.ownerUserId)}</Info>
              <Info label={a.address}>{account.address}</Info>
            </dl>
            {account.notes ? (
              <p className="mt-4 text-sm whitespace-pre-line text-muted-foreground">{account.notes}</p>
            ) : null}
          </Panel>
        }
      >
        <Section title={a.contacts}>
          <ContactsTable
            contacts={contacts}
            accounts={[{ value: account.id, label: account.name }]}
            caps={caps}
            accountId={account.id}
            compact
          />
        </Section>
        <Panel title={a.deals} flush>
          <DealRows deals={deals} empty={ss.leads.noDeals} />
        </Panel>
      </DetailLayout>
    </>
  );
}
