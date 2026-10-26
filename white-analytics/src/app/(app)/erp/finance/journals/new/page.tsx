import type { Metadata } from "next";
import { PageHeader } from "@/components/dashboard/page-header";
import { requireErp } from "@/features/erp/access";
import { load } from "@/features/erp/shared/load";
import { ErpLoadError } from "@/features/erp/shared/states";
import { JournalForm } from "@/features/erp/finance/components/journal-form";
import { financePerms } from "@/features/erp/finance/perms";
import { listPostableAccounts } from "@/features/erp/finance/queries";
import { fs } from "@/features/erp/finance/strings";

export const metadata: Metadata = { title: fs.journals.newTitle };

export default async function NewJournalPage() {
  const { access } = await requireErp("finance.write");
  const accounts = await load(listPostableAccounts);
  return (
    <>
      <PageHeader eyebrow={fs.journals.title} title={fs.journals.newTitle} />
      {accounts.error ? (
        <ErpLoadError error={accounts.error} />
      ) : (
        <JournalForm accounts={accounts.data} canApprove={financePerms(access).canApprove} />
      )}
    </>
  );
}
