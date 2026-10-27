import type { Metadata } from "next";
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
        back={{ href: "/erp/me/reimbursements", label: fs.back }}
      />
      {res.error ? <ErpLoadError error={res.error} /> : <ClaimDetail claim={res.data} mode="mine" />}
    </>
  );
}
