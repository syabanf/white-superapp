import type { Metadata } from "next";
import { requireErp } from "@/features/erp/access";
import { LedgerKindPage } from "@/features/erp/finance/components/ledger-page";
import { fs } from "@/features/erp/finance/strings";

export const metadata: Metadata = { title: fs.invoices.AP.title };

export default async function PayablesPage(props: PageProps<"/erp/finance/payables">) {
  const sp = await props.searchParams;
  const { access, offline } = await requireErp("finance.read");
  return <LedgerKindPage kind="AP" sp={sp} access={access} offline={offline} />;
}
