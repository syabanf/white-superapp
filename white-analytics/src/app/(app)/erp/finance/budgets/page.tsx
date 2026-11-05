import type { Metadata } from "next";
import { PageHeader } from "@/components/dashboard/page-header";
import { requireErp } from "@/features/erp/access";
import { BudgetEditor } from "@/features/erp/finance/components/budget-editor";
import { financePerms } from "@/features/erp/finance/perms";
import { getBudgetYear, listPostableAccounts } from "@/features/erp/finance/queries";
import { fs } from "@/features/erp/finance/strings";
import { OfflineNotice } from "@/features/erp/offline-notice";
import { load, sp1 } from "@/features/erp/shared/load";
import { ErpLoadError } from "@/features/erp/shared/states";
import { UrlTabs } from "@/features/erp/shared/url-tabs";

const s = fs.budgets;
const PL_TYPES = ["REVENUE", "COGS", "EXPENSE", "OTHER_INCOME", "OTHER_EXPENSE"];
export const metadata: Metadata = { title: s.title };

export default async function BudgetsPage(props: PageProps<"/erp/finance/budgets">) {
  const sp = await props.searchParams;
  const { access, offline } = await requireErp("finance.read");
  const header = (
    <PageHeader back={{ href: "/erp/finance/reports?r=budget-vs-actual", label: s.back }} eyebrow={fs.eyebrow} title={s.title} description={s.subtitle} />
  );
  if (offline) return <>{header}<OfflineNotice message={offline.message} /></>;

  const thisYear = new Date().getFullYear();
  const year = Number(sp1(sp.year)) || thisYear;
  const [entries, accounts] = await Promise.all([load(() => getBudgetYear(year)), load(listPostableAccounts)]);
  const tabs = (
    <UrlTabs
      param="year"
      label={s.year}
      // UrlTabs marks the first tab when the URL has no year, so this year goes first.
      tabs={[thisYear, thisYear + 1, thisYear - 1].map((y) => ({ value: String(y), label: String(y) }))}
    />
  );
  const failed = entries.error ?? accounts.error;
  return (
    <>
      {header}
      {tabs}
      {failed ? (
        <ErpLoadError error={failed} />
      ) : (
        <BudgetEditor
          key={year}
          year={year}
          entries={entries.data ?? []}
          accounts={(accounts.data ?? []).filter((a) => PL_TYPES.includes(a.accountTypeCode))}
          canWrite={financePerms(access).canWrite}
        />
      )}
    </>
  );
}
