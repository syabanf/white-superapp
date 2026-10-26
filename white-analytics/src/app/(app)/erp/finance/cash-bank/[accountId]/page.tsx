import type { Metadata } from "next";
import Link from "next/link";
import { ArrowLeft } from "lucide-react";
import { Button } from "@/components/ui/button";
import { PageHeader } from "@/components/dashboard/page-header";
import { requireErp } from "@/features/erp/access";
import { load, sp1 } from "@/features/erp/shared/load";
import { DateRangeFilter } from "@/features/erp/shared/period-picker";
import { PrintButton } from "@/features/erp/shared/print";
import { ErpLoadError } from "@/features/erp/shared/states";
import { LedgerTable } from "@/features/erp/finance/components/ledger-table";
import { getCashBankLedger } from "@/features/erp/finance/queries";
import { fs } from "@/features/erp/finance/strings";
import { formatDate } from "@/lib/format";

export const metadata: Metadata = { title: fs.cash.ledger };

export default async function CashLedgerPage(props: PageProps<"/erp/finance/cash-bank/[accountId]">) {
  const { accountId } = await props.params;
  const sp = await props.searchParams;
  await requireErp("finance.read");
  const res = await load(() => getCashBankLedger(accountId, { dateFrom: sp1(sp.from), dateTo: sp1(sp.to) }));
  const acc = res.data?.account;
  return (
    <>
      <PageHeader
        eyebrow={fs.cash.ledger}
        title={acc ? `${acc.codeDisplay || acc.code} ${acc.name}` : fs.cash.ledger}
        description={res.data ? `${formatDate(res.data.dateFrom)} – ${formatDate(res.data.dateTo)}` : undefined}
        actions={
          <>
            <Button asChild size="sm" variant="ghost" data-print-hide>
              <Link href="/erp/finance/cash-bank">
                <ArrowLeft className="size-4" /> {fs.back}
              </Link>
            </Button>
            <DateRangeFilter />
            <PrintButton label={fs.print} />
          </>
        }
      />
      {res.error ? <ErpLoadError error={res.error} /> : <LedgerTable ledger={res.data} />}
    </>
  );
}
