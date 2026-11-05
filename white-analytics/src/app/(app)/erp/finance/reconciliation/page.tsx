import type { Metadata } from "next";
import { KpiTile } from "@/components/dashboard/kpi-tile";
import { PageHeader } from "@/components/dashboard/page-header";
import { requireErp } from "@/features/erp/access";
import { RematchButton, StatementLines, StatementUpload } from "@/features/erp/finance/components/reconciliation";
import { financePerms } from "@/features/erp/finance/perms";
import { listPostableAccounts, listReconAccounts, listStatementLines } from "@/features/erp/finance/queries";
import { fs } from "@/features/erp/finance/strings";
import { OfflineNotice } from "@/features/erp/offline-notice";
import { load, sp1, spInt } from "@/features/erp/shared/load";
import { formatRupiah } from "@/features/erp/shared/money";
import { ErpLoadError } from "@/features/erp/shared/states";
import { UrlTabs } from "@/features/erp/shared/url-tabs";

const s = fs.recon;
export const metadata: Metadata = { title: s.title };

export default async function ReconciliationPage(props: PageProps<"/erp/finance/reconciliation">) {
  const sp = await props.searchParams;
  const { access, offline } = await requireErp("finance.read");
  const { canWrite, canApprove } = financePerms(access);
  const header = (actions?: React.ReactNode) => <PageHeader eyebrow={fs.eyebrow} title={s.title} description={s.subtitle} actions={actions} />;
  if (offline) return <>{header()}<OfflineNotice message={offline.message} /></>;

  const accounts = await load(listReconAccounts);
  if (accounts.error) return <>{header()}<ErpLoadError error={accounts.error} /></>;
  if (accounts.data.length === 0) return <>{header()}<p className="text-sm text-muted-foreground">{s.noAccounts}</p></>;

  const selected = accounts.data.find((a) => a.id === sp1(sp.account)) ?? accounts.data[0];
  const page = spInt(sp.page, 1);
  const pageSize = spInt(sp.pageSize, 25);
  const [lines, postable] = await Promise.all([
    load(() => listStatementLines({ accountId: selected.id, status: sp1(sp.status), dateFrom: sp1(sp.from), dateTo: sp1(sp.to), page, pageSize })),
    canApprove ? load(listPostableAccounts) : null,
  ]);
  return (
    <>
      {header(
        canWrite ? (
          <>
            <RematchButton accountId={selected.id} />
            <StatementUpload accountId={selected.id} />
          </>
        ) : null,
      )}
      <div className="max-w-full overflow-x-auto">
        <UrlTabs
          param="account"
          label={s.account}
          tabs={accounts.data.map((a) => ({ value: a.id, label: a.isCreditCard ? `${a.name} · ${s.card}` : a.name }))}
          reset={["page", "status", "from", "to"]}
        />
      </div>
      <div className="grid grid-cols-2 gap-(--gap) md:grid-cols-4">
        <KpiTile label={s.book} value={formatRupiah(selected.bookBalance)} size="sm" />
        <KpiTile label={s.statuses.UNMATCHED} value={String(selected.unmatched)} size="sm" />
        <KpiTile label={s.statuses.SUGGESTED} value={String(selected.suggested)} size="sm" />
        <KpiTile label={s.statuses.MATCHED} value={String(selected.matched)} size="sm" />
      </div>
      {lines.error ? (
        <ErpLoadError error={lines.error} />
      ) : (
        <StatementLines
          lines={lines.data.lines ?? []}
          page={{ page, pageSize, total: lines.data.total }}
          accounts={postable?.data ?? []}
          canWrite={canWrite}
          canApprove={canApprove}
        />
      )}
    </>
  );
}
