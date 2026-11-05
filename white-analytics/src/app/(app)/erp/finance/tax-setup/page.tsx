import type { Metadata } from "next";
import { PageHeader } from "@/components/dashboard/page-header";
import { requireErp } from "@/features/erp/access";
import { FiscalDialog, FiscalTable, ProfileDialog, ProfilesTable } from "@/features/erp/finance/components/tax-parties";
import { ChargeDialog, ChargeTable, TaxDialog, TaxTable } from "@/features/erp/finance/components/tax-setup";
import { financePerms } from "@/features/erp/finance/perms";
import { listChargeTypes, listFiscalPositions, listPartyProfiles, listPostableAccounts, listTaxCodes } from "@/features/erp/finance/queries";
import { fs } from "@/features/erp/finance/strings";
import { OfflineNotice } from "@/features/erp/offline-notice";
import { matchesActive, matchesText } from "@/features/erp/shared/list-filter";
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

  const [q, status] = [sp1(sp.q), sp1(sp.status)];
  const tab = sp1(sp.tab);
  const tabs = (
    <UrlTabs
      label={s.title}
      tabs={[
        { value: "taxes", label: s.tabTaxes },
        { value: "charges", label: s.tabCharges },
        { value: "fiscal", label: s.tabFiscal },
        { value: "profiles", label: s.tabProfiles },
      ]}
      reset={["q", "status", "kind", "scope", "computation", "side", "pkp"]}
    />
  );

  if (tab === "fiscal") {
    const res = await load(() => Promise.all([listFiscalPositions(), listTaxCodes()]));
    return (
      <>
        {header(canApprove && res.data ? <FiscalDialog taxes={res.data[1]} /> : null)}
        {tabs}
        {res.error ? (
          <ErpLoadError error={res.error} />
        ) : (
          <FiscalTable rows={res.data[0].filter((f) => matchesText(q, f.name, f.notes) && matchesActive(status, f.isActive))} taxes={res.data[1]} canApprove={canApprove} />
        )}
      </>
    );
  }
  if (tab === "profiles") {
    const [side, pkp] = [sp1(sp.side), sp1(sp.pkp)];
    const res = await load(() => Promise.all([listPartyProfiles(), listFiscalPositions()]));
    return (
      <>
        {header(canApprove && res.data ? <ProfileDialog positions={res.data[1]} /> : null)}
        {tabs}
        {res.error ? (
          <ErpLoadError error={res.error} />
        ) : (
          <ProfilesTable
            rows={res.data[0].filter((p) => matchesText(q, p.name, p.npwp) && (!side || p.side === side) && (!pkp || (pkp === "yes") === p.isPkp))}
            positions={res.data[1]}
            canApprove={canApprove}
          />
        )}
      </>
    );
  }

  const accounts = (await load(listPostableAccounts)).data ?? [];
  const scope = sp1(sp.scope);
  if (tab === "charges") {
    const computation = sp1(sp.computation);
    const res = await load(() => listChargeTypes());
    return (
      <>
        {header(canApprove ? <ChargeDialog accounts={accounts} /> : null)}
        {tabs}
        {res.error ? (
          <ErpLoadError error={res.error} />
        ) : (
          <ChargeTable
            charges={res.data.filter((c) => matchesText(q, c.code, c.name) && matchesActive(status, c.isActive) && (!scope || c.scope === scope) && (!computation || c.computation === computation))}
            accounts={accounts}
            canApprove={canApprove}
          />
        )}
      </>
    );
  }
  const kind = sp1(sp.kind);
  const res = await load(() => listTaxCodes());
  return (
    <>
      {header(canApprove ? <TaxDialog accounts={accounts} /> : null)}
      {tabs}
      {res.error ? (
        <ErpLoadError error={res.error} />
      ) : (
        <TaxTable
          taxes={res.data.filter((t) => matchesText(q, t.code, t.name) && matchesActive(status, t.isActive) && (!kind || t.kind === kind) && (!scope || t.scope === scope))}
          accounts={accounts}
          canApprove={canApprove}
        />
      )}
    </>
  );
}
