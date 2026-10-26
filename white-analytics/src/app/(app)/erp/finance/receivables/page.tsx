import type { Metadata } from "next";
import { requireErp } from "@/features/erp/access";
import { LedgerKindPage } from "@/features/erp/finance/components/ledger-page";
import { fs } from "@/features/erp/finance/strings";

export const metadata: Metadata = { title: fs.invoices.AR.title };

export default async function ReceivablesPage(props: PageProps<"/erp/finance/receivables">) {
  const sp = await props.searchParams;
  const { access, offline } = await requireErp("finance.read");
  return <LedgerKindPage kind="AR" sp={sp} access={access} offline={offline} />;
}
