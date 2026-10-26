import type { Metadata } from "next";
import Link from "next/link";
import { ArrowLeft } from "lucide-react";
import { Button } from "@/components/ui/button";
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
        actions={
          <Button asChild size="sm" variant="ghost">
            <Link href="/erp/finance/fiscal">
              <ArrowLeft className="size-4" /> {fs.back}
            </Link>
          </Button>
        }
      />
      {res.error ? (
        <ErpLoadError error={res.error} />
      ) : (
        <BeginningBalanceEditor data={res.data} canWrite={perms.canWrite} canApprove={perms.canApprove} />
      )}
    </>
  );
}
