import type { Metadata } from "next";
import { Alert, AlertTitle } from "@/components/ui/alert";
import { PageHeader } from "@/components/dashboard/page-header";
import { requireErp } from "@/features/erp/access";
import { load } from "@/features/erp/shared/load";
import { ErpLoadError } from "@/features/erp/shared/states";
import { JournalForm } from "@/features/erp/finance/components/journal-form";
import { financePerms } from "@/features/erp/finance/perms";
import { getJournalEntry, listPostableAccounts } from "@/features/erp/finance/queries";
import { fs } from "@/features/erp/finance/strings";

export const metadata: Metadata = { title: fs.journals.editTitle };

export default async function EditJournalPage(props: PageProps<"/erp/finance/journals/[id]/edit">) {
  const { id } = await props.params;
  const { access } = await requireErp("finance.write");
  const [entry, accounts] = await Promise.all([load(() => getJournalEntry(id)), load(listPostableAccounts)]);
  const error = entry.error ?? accounts.error;
  return (
    <>
      <PageHeader eyebrow={fs.journals.title} title={`${fs.journals.editTitle} ${entry.data?.entryNo ?? ""}`} />
      {error ? (
        <ErpLoadError error={error} />
      ) : entry.data && accounts.data && entry.data.canEdit && entry.data.status === "DRAFT" ? (
        <JournalForm entry={entry.data} accounts={accounts.data} canApprove={financePerms(access).canApprove} />
      ) : (
        <Alert className="rounded-2xl">
          <AlertTitle>{fs.journals.readonly}</AlertTitle>
        </Alert>
      )}
    </>
  );
}
