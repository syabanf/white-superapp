import type { Metadata } from "next";
import { PageHeader } from "@/components/dashboard/page-header";
import { requireErp } from "@/features/erp/access";
import { OfflineNotice } from "@/features/erp/offline-notice";
import { load, sp1, spInt } from "@/features/erp/shared/load";
import { ErpLoadError } from "@/features/erp/shared/states";
import { UrlTabs } from "@/features/erp/shared/url-tabs";
import {
  CashAccountCards,
  CashMovementDialog,
  CashTransferDialog,
  MovementsTable,
  TransfersTable,
} from "@/features/erp/finance/components/cash-bank";
import { financePerms } from "@/features/erp/finance/perms";
import {
  listCashBankAccounts,
  listCashMovements,
  listCashTransfers,
  listPostableAccounts,
} from "@/features/erp/finance/queries";
import { fs } from "@/features/erp/finance/strings";

export const metadata: Metadata = { title: fs.cash.title };

const c = fs.cash;

export default async function CashBankPage(props: PageProps<"/erp/finance/cash-bank">) {
  const sp = await props.searchParams;
  const { access, offline } = await requireErp("finance.read");
  const { canWrite } = financePerms(access);
  const tab = sp1(sp.tab) ?? "accounts";
  const postable = canWrite ? await load(listPostableAccounts) : null;
  const accounts = postable?.data ?? [];
  const header = (
    <PageHeader
      eyebrow={fs.eyebrow}
      title={c.title}
      description={c.subtitle}
      actions={
        canWrite && !offline ? (
          <>
            <CashMovementDialog kind="CASH_IN" accounts={accounts} />
            <CashMovementDialog kind="CASH_OUT" accounts={accounts} />
            <CashTransferDialog accounts={accounts} />
          </>
        ) : null
      }
    />
  );
  const tabs = (
    <UrlTabs
      label={c.title}
      tabs={[
        { value: "accounts", label: c.tabAccounts },
        { value: "in", label: c.tabIn },
        { value: "out", label: c.tabOut },
        { value: "transfers", label: c.tabTransfers },
      ]}
      reset={["page", "q", "from", "to"]}
    />
  );
  if (offline) return <>{header}<OfflineNotice message={offline.message} /></>;

  const page = spInt(sp.page, 1);
  const pageSize = spInt(sp.pageSize, 20);
  const q = { search: sp1(sp.q), dateFrom: sp1(sp.from), dateTo: sp1(sp.to), page, pageSize };
  let body: React.ReactNode;
  if (tab === "in" || tab === "out") {
    const kind = tab === "in" ? "CASH_IN" : "CASH_OUT";
    const res = await load(() => listCashMovements({ ...q, kind }));
    body = res.error ? (
      <ErpLoadError error={res.error} />
    ) : (
      <MovementsTable rows={res.data.movements} kind={kind} page={{ page, pageSize, total: res.data.total }} />
    );
  } else if (tab === "transfers") {
    const res = await load(() => listCashTransfers(q));
    body = res.error ? (
      <ErpLoadError error={res.error} />
    ) : (
      <TransfersTable rows={res.data.transfers} page={{ page, pageSize, total: res.data.total }} />
    );
  } else {
    const res = await load(listCashBankAccounts);
    body = res.error ? (
      <ErpLoadError error={res.error} />
    ) : (
      <CashAccountCards accounts={res.data.accounts} total={res.data.totalBalance} />
    );
  }
  return (
    <>
      {header}
      {tabs}
      {body}
    </>
  );
}
