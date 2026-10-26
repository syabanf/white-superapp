import type { Metadata } from "next";
import Link from "next/link";
import { ArrowLeft } from "lucide-react";
import { Button } from "@/components/ui/button";
import { PageHeader } from "@/components/dashboard/page-header";
import { requireErp } from "@/features/erp/access";
import { load } from "@/features/erp/shared/load";
import { ErpLoadError } from "@/features/erp/shared/states";
import { ErpStatusBadge } from "@/features/erp/shared/status-badge";
import { JournalActions, JournalDetail } from "@/features/erp/finance/components/journal-detail";
import { financePerms } from "@/features/erp/finance/perms";
import { getJournalEntry } from "@/features/erp/finance/queries";
import { fs } from "@/features/erp/finance/strings";

export const metadata: Metadata = { title: fs.journals.title };

export default async function JournalDetailPage(props: PageProps<"/erp/finance/journals/[id]">) {
  const { id } = await props.params;
  const { access } = await requireErp("finance.read");
  const res = await load(() => getJournalEntry(id));
  const back = (
    <Button asChild size="sm" variant="ghost">
      <Link href="/erp/finance/journals">
        <ArrowLeft className="size-4" /> {fs.back}
      </Link>
    </Button>
  );
  if (res.error) return <>{back}<ErpLoadError error={res.error} /></>;
  const entry = res.data;
  return (
    <>
      <PageHeader
        eyebrow={fs.journals.title}
        title={
          <span className="flex items-center gap-3">
            {entry.entryNo || fs.journals.title}
            <ErpStatusBadge status={entry.status} />
          </span>
        }
        actions={
          <>
            {back}
            <JournalActions entry={entry} perms={financePerms(access)} />
          </>
        }
      />
      <JournalDetail entry={entry} />
    </>
  );
}
