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

export const metadata: Metadata = { title: fs.reimburse.myTitle };

export default async function MyReimbursementDetailPage(props: PageProps<"/erp/me/reimbursements/[id]">) {
  const { id } = await props.params;
  await requireErp("self.service");
  const res = await load(() => getClaim(id));
  return (
    <>
      <PageHeader
        eyebrow={fs.reimburse.myTitle}
        title={res.data?.claimNo || fs.reimburse.myTitle}
        actions={
          <Button asChild size="sm" variant="ghost">
            <Link href="/erp/me/reimbursements">
              <ArrowLeft className="size-4" /> {fs.back}
            </Link>
          </Button>
        }
      />
      {res.error ? <ErpLoadError error={res.error} /> : <ClaimDetail claim={res.data} mode="mine" />}
    </>
  );
}
