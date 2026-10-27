import type { Metadata } from "next";
import { PageHeader } from "@/components/dashboard/page-header";
import { requireErp } from "@/features/erp/access";
import { load } from "@/features/erp/shared/load";
import { ErpLoadError } from "@/features/erp/shared/states";
import { BeginningBalanceEditor } from "@/features/erp/finance/components/beginning-balance";
import { financePerms } from "@/features/erp/finance/perms";
import { getBeginningBalance } from "@/features/erp/finance/queries";
import { fs } from "@/features/erp/finance/strings";

export const metadata: Metadata = { title: fs.fiscal.bb.title };

export default async function BeginningBalancePage(props: PageProps<"/erp/finance/fiscal/[id]/beginning-balance">) {
  const { id } = await props.params;
  const { access } = await requireErp("finance.read");
  const perms = financePerms(access);
  const res = await load(() => getBeginningBalance(id));
  return (
    <>
      <PageHeader
        eyebrow={fs.fiscal.title}
        title={fs.fiscal.bb.title}
        description={res.data ? fs.fiscal.bb.subtitle(res.data.fiscalYearName) : undefined}
        back={{ href: "/erp/finance/fiscal", label: fs.back }}
      />
      {res.error ? (
        <ErpLoadError error={res.error} />
      ) : (
        <BeginningBalanceEditor data={res.data} canWrite={perms.canWrite} canApprove={perms.canApprove} />
      )}
    </>
  );
}
