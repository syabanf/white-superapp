import type { Metadata } from "next";
import Link from "next/link";
import { Inbox } from "lucide-react";
import { Alert, AlertDescription, AlertTitle } from "@/components/ui/alert";
import { Button } from "@/components/ui/button";
import { PageHeader } from "@/components/dashboard/page-header";
import { requireErp } from "@/features/erp/access";
import { OfflineNotice } from "@/features/erp/offline-notice";
import { load, sp1 } from "@/features/erp/shared/load";
import { ErpLoadError } from "@/features/erp/shared/states";
import { NewClaimButton } from "@/features/erp/finance/components/claim-detail";
import { MyClaimsTable } from "@/features/erp/finance/components/claims";
import { listApprovalInbox, listMyClaims } from "@/features/erp/finance/queries";
import { fs } from "@/features/erp/finance/strings";

export const metadata: Metadata = { title: fs.reimburse.myTitle };

const rb = fs.reimburse;

export default async function MyReimbursementsPage(props: PageProps<"/erp/me/reimbursements">) {
  const sp = await props.searchParams;
  const { offline } = await requireErp("self.service");
  const header = (
    <PageHeader eyebrow={fs.eyebrow} title={rb.myTitle} description={rb.mySubtitle} actions={offline ? null : <NewClaimButton />} />
  );
  if (offline) return <>{header}<OfflineNotice message={offline.message} /></>;
  const [mine, inbox] = await Promise.all([load(() => listMyClaims(sp1(sp.status))), load(listApprovalInbox)]);
  const teamCount = inbox.data?.claims.length ?? 0;
  return (
    <>
      {header}
      {teamCount > 0 ? (
        <Alert className="rounded-2xl">
          <Inbox className="size-4" />
          <AlertTitle>{rb.teamInbox}</AlertTitle>
          <AlertDescription className="flex flex-wrap items-center justify-between gap-2">
            <span>{rb.teamInboxDesc(teamCount)}</span>
            <Button asChild size="xs" variant="outline">
              <Link href="/erp/finance/reimbursements">{rb.tabInbox}</Link>
            </Button>
          </AlertDescription>
        </Alert>
      ) : null}
      {mine.error ? <ErpLoadError error={mine.error} /> : <MyClaimsTable claims={mine.data.claims} />}
    </>
  );
}
