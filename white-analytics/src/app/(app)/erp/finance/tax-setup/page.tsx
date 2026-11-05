import type { Metadata } from "next";
import { PageHeader } from "@/components/dashboard/page-header";
import { requireErp } from "@/features/erp/access";
import { ChargeDialog, ChargeTable, TaxDialog, TaxTable } from "@/features/erp/finance/components/tax-setup";
import { financePerms } from "@/features/erp/finance/perms";
import { listChargeTypes, listPostableAccounts, listTaxCodes } from "@/features/erp/finance/queries";
import { fs } from "@/features/erp/finance/strings";
import { OfflineNotice } from "@/features/erp/offline-notice";
import { load, sp1 } from "@/features/erp/shared/load";
import { ErpLoadError } from "@/features/erp/shared/states";
import { UrlTabs } from "@/features/erp/shared/url-tabs";

const s = fs.taxSetup;
export const metadata: Metadata = { title: s.title };

export default async function TaxSetupPage(props: PageProps<"/erp/finance/tax-setup">) {
  const sp = await props.searchParams;
  const { access, offline } = await requireErp("finance.read");
  const { canApprove } = financePerms(access);
  const header = (actions?: React.ReactNode) => <PageHeader eyebrow={fs.eyebrow} title={s.title} description={s.subtitle} actions={actions} />;
  if (offline) return <>{header()}<OfflineNotice message={offline.message} /></>;

  const charges = sp1(sp.tab) === "charges";
  const accounts = (await load(listPostableAccounts)).data ?? [];
  const tabs = (
    <UrlTabs
      label={s.title}
      tabs={[
        { value: "taxes", label: s.tabTaxes },
        { value: "charges", label: s.tabCharges },
      ]}
    />
  );
  if (charges) {
    const res = await load(() => listChargeTypes());
    return (
      <>
        {header(canApprove ? <ChargeDialog accounts={accounts} /> : null)}
        {tabs}
        {res.error ? <ErpLoadError error={res.error} /> : <ChargeTable charges={res.data} accounts={accounts} canApprove={canApprove} />}
      </>
    );
  }
  const res = await load(() => listTaxCodes());
  return (
    <>
      {header(canApprove ? <TaxDialog accounts={accounts} /> : null)}
      {tabs}
      {res.error ? <ErpLoadError error={res.error} /> : <TaxTable taxes={res.data} accounts={accounts} canApprove={canApprove} />}
    </>
  );
}
