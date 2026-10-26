import { PageHeader } from "@/components/dashboard/page-header";
import type { ErpAccess } from "@/components/shell/nav-config";
import type { GatewayError } from "@/lib/erp/gateway";
import { OfflineNotice } from "@/features/erp/offline-notice";
import { load, sp1, spInt } from "@/features/erp/shared/load";
import { DateFilter } from "@/features/erp/shared/period-picker";
import { ErpLoadError } from "@/features/erp/shared/states";
import { UrlTabs } from "@/features/erp/shared/url-tabs";
import { financePerms } from "../perms";
import { getAging, listInvoices, listOpenInvoices, listPayments, listPostableAccounts } from "../queries";
import type { LedgerKind } from "../types";
import { fs } from "../strings";
import { InvoicesTable } from "./invoices";
import { AgingView, PaymentDialog, PaymentsTable } from "./payments";

type SP = Record<string, string | string[] | undefined>;

/** Payables and receivables share one page: invoices, payments/receipts, aging. */
export async function LedgerKindPage({
  kind,
  sp,
  access,
  offline,
}: {
  kind: LedgerKind;
  sp: SP;
  access: ErpAccess;
  offline: GatewayError | null;
}) {
  const k = fs.invoices[kind];
  const perms = financePerms(access);
  const tab = sp1(sp.tab) ?? "invoices";
  const tabs = (
    <UrlTabs
      label={k.title}
      tabs={[
        { value: "invoices", label: fs.invoices.tabInvoices },
        { value: "payments", label: k.tabPayments },
        { value: "aging", label: fs.invoices.tabAging },
      ]}
      reset={["page", "q", "status", "pay", "asOf"]}
    />
  );
  const header = (actions?: React.ReactNode) => (
    <PageHeader eyebrow={fs.eyebrow} title={k.title} description={k.subtitle} actions={actions} />
  );
  if (offline) return <>{header()}<OfflineNotice message={offline.message} /></>;

  const page = spInt(sp.page, 1);
  const pageSize = spInt(sp.pageSize, 20);

  if (tab === "aging") {
    const res = await load(() => getAging(kind, sp1(sp.asOf)));
    return (
      <>
        {header(<DateFilter param="asOf" label={fs.invoices.asOf} />)}
        {tabs}
        {res.error ? <ErpLoadError error={res.error} /> : <AgingView kind={kind} aging={res.data} />}
      </>
    );
  }

  if (tab === "payments") {
    const [res, extra] = await Promise.all([
      load(() => listPayments(kind, { search: sp1(sp.q), page, pageSize })),
      perms.canApprove ? load(() => Promise.all([listOpenInvoices(kind), listPostableAccounts()])) : null,
    ]);
    return (
      <>
        {header(extra?.data ? <PaymentDialog kind={kind} openInvoices={extra.data[0]} accounts={extra.data[1]} /> : null)}
        {tabs}
        {res.error ? (
          <ErpLoadError error={res.error} />
        ) : (
          <PaymentsTable kind={kind} payments={res.data.payments} page={{ page, pageSize, total: res.data.total }} perms={perms} />
        )}
      </>
    );
  }

  const [res, accounts] = await Promise.all([
    load(() =>
      listInvoices(kind, { search: sp1(sp.q), status: sp1(sp.status), paymentStatus: sp1(sp.pay), page, pageSize }),
    ),
    perms.canWrite ? load(listPostableAccounts) : null,
  ]);
  return (
    <>
      {header()}
      {tabs}
      {res.error ? (
        <ErpLoadError error={res.error} />
      ) : (
        <InvoicesTable
          kind={kind}
          invoices={res.data.invoices}
          page={{ page, pageSize, total: res.data.total }}
          accounts={accounts?.data ?? []}
          perms={perms}
        />
      )}
    </>
  );
}
