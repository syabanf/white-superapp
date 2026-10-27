import type { Metadata } from "next";
import Link from "next/link";
import { Plus } from "lucide-react";
import { Button } from "@/components/ui/button";
import { PageHeader } from "@/components/dashboard/page-header";
import { requireErp } from "@/features/erp/access";
import { OfflineNotice } from "@/features/erp/offline-notice";
import { load, sp1, spInt } from "@/features/erp/shared/load";
import { ErpLoadError } from "@/features/erp/shared/states";
import { UrlTabs } from "@/features/erp/shared/url-tabs";
import { JournalsTable } from "@/features/erp/finance/components/journals-table";
import { PostingLogs } from "@/features/erp/finance/components/posting-logs";
import { financePerms } from "@/features/erp/finance/perms";
import { listJournalEntries, listPostingLogs } from "@/features/erp/finance/queries";
import { fs } from "@/features/erp/finance/strings";

export const metadata: Metadata = { title: fs.journals.title };

export default async function JournalsPage(props: PageProps<"/erp/finance/journals">) {
  const sp = await props.searchParams;
  const { access, offline } = await requireErp("finance.read");
  const perms = financePerms(access);
  const tab = sp1(sp.tab) === "logs" ? "logs" : "entries";
  const header = (
    <PageHeader
      eyebrow={fs.eyebrow}
      title={fs.journals.title}
      description={fs.journals.subtitle}
      actions={
        perms.canWrite ? (
          <Button asChild>
            <Link href="/erp/finance/journals/new">
              <Plus className="size-4" /> {fs.journals.new}
            </Link>
          </Button>
        ) : null
      }
    />
  );
  const tabs = (
    <UrlTabs
      label={fs.journals.title}
      tabs={[
        { value: "entries", label: fs.journals.tabEntries },
        { value: "logs", label: fs.journals.tabLogs },
      ]}
      reset={["page", "q", "status", "type", "from", "to", "logStatus"]}
    />
  );
  if (offline) return <>{header}<OfflineNotice message={offline.message} /></>;

  if (tab === "logs") {
    const res = await load(() => listPostingLogs({ status: sp1(sp.logStatus) }));
    return (
      <>
        {header}
        {tabs}
        {res.error ? <ErpLoadError error={res.error} /> : <PostingLogs logs={res.data} canApprove={perms.canApprove} />}
      </>
    );
  }

  const page = spInt(sp.page, 1);
  const pageSize = spInt(sp.pageSize, 20);
  const res = await load(() =>
    listJournalEntries({
      search: sp1(sp.q),
      status: sp1(sp.status),
      entryType: sp1(sp.type),
      dateFrom: sp1(sp.from),
      dateTo: sp1(sp.to),
      accountId: sp1(sp.account),
      page,
      pageSize,
    }),
  );
  return (
    <>
      {header}
      {tabs}
      {res.error ? (
        <ErpLoadError error={res.error} />
      ) : (
        <JournalsTable entries={res.data.entries} page={{ page, pageSize, total: res.data.total }} />
      )}
    </>
  );
}
