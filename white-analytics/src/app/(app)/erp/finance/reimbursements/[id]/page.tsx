import type { Metadata } from "next";
import Link from "next/link";
import { ArrowLeft } from "lucide-react";
import { Button } from "@/components/ui/button";
import { PageHeader } from "@/components/dashboard/page-header";
import { requireErp } from "@/features/erp/access";
import { load } from "@/features/erp/shared/load";
import { ErpLoadError } from "@/features/erp/shared/states";
import { ClaimDetail } from "@/features/erp/finance/components/claim-detail";
import { getClaim } from "@/features/erp/finance/queries";
import { fs } from "@/features/erp/finance/strings";

export const metadata: Metadata = { title: fs.reimburse.title };

/** Review view of a claim; the backend allows owners, approvers and finance.read. */
export default async function ReimbursementDetailPage(props: PageProps<"/erp/finance/reimbursements/[id]">) {
  const { id } = await props.params;
  await requireErp();
  const res = await load(() => getClaim(id));
  return (
    <>
      <PageHeader
        eyebrow={fs.reimburse.title}
        title={res.data?.claimNo || fs.reimburse.title}
        actions={
          <Button asChild size="sm" variant="ghost">
            <Link href="/erp/finance/reimbursements">
              <ArrowLeft className="size-4" /> {fs.back}
            </Link>
          </Button>
        }
      />
      {res.error ? <ErpLoadError error={res.error} /> : <ClaimDetail claim={res.data} mode="review" />}
    </>
  );
}
