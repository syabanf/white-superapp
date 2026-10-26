import type { Metadata } from "next";
import Link from "next/link";
import { notFound } from "next/navigation";
import { ChevronLeft } from "lucide-react";
import { PageHeader } from "@/components/dashboard/page-header";
import { requireErp } from "@/features/erp/access";
import { OfflineNotice } from "@/features/erp/offline-notice";
import { load, sp1 } from "@/features/erp/shared/load";
import { ErpError } from "@/features/erp/shared/states";
import { QuotationEditor } from "@/features/erp/sales/components/quotation-editor";
import { getDeal, listCatalog, listRetainerDiscounts } from "@/features/erp/sales/queries";
import { ss } from "@/features/erp/sales/strings";

const s = ss.quotations;
export const metadata: Metadata = { title: s.newTitle };

export default async function NewQuotationPage(props: PageProps<"/erp/sales/quotations/new">) {
  const sp = await props.searchParams;
  const dealId = sp1(sp.deal);
  if (!dealId) notFound();
  const { offline } = await requireErp("sales.write");
  if (offline) return <OfflineNotice message={offline.message} />;
  const [deal, catalog, retainer] = await Promise.all([load(() => getDeal(dealId)), load(() => listCatalog()), load(listRetainerDiscounts)]);
  if (!deal.data) return <ErpError message={deal.error?.message} />;
  const d = deal.data.deal;
  return (
    <>
      <Link href={`/erp/sales/pipeline/${d.id}`} className="-mb-3 inline-flex w-fit items-center gap-1 text-sm text-muted-foreground hover:text-foreground">
        <ChevronLeft className="size-4" /> {d.title}
      </Link>
      <PageHeader eyebrow={s.title} title={s.newTitle} description={d.clientName} />
      <QuotationEditor quotation={null} dealId={d.id} catalog={catalog.data ?? []} retainerRules={retainer.data ?? []} />
    </>
  );
}
