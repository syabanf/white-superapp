import type { Metadata } from "next";
import { PageHeader } from "@/components/dashboard/page-header";
import { requireErp } from "@/features/erp/access";
import { OfflineNotice } from "@/features/erp/offline-notice";
import { load, sp1 } from "@/features/erp/shared/load";
import { ErpError } from "@/features/erp/shared/states";
import { QuotationDealPicker } from "@/features/erp/sales/components/quotation-deal-picker";
import { QuotationEditor } from "@/features/erp/sales/components/quotation-editor";
import { getDocumentParty, listCatalog, listDeals, listRetainerDiscounts } from "@/features/erp/sales/queries";
import { ss } from "@/features/erp/sales/strings";

const s = ss.quotations;
export const metadata: Metadata = { title: s.newTitle };

export default async function NewQuotationPage(props: PageProps<"/erp/sales/quotations/new">) {
  const sp = await props.searchParams;
  const dealId = sp1(sp.deal);
  const { offline } = await requireErp("sales.write");
  if (offline) return <OfflineNotice message={offline.message} />;
  const [deals, party, catalog, retainer] = await Promise.all([
    load(() => listDeals({ status: "open", pageSize: 200 })),
    dealId ? load(() => getDocumentParty(dealId)) : null,
    dealId ? load(() => listCatalog()) : null,
    dealId ? load(listRetainerDiscounts) : null,
  ]);
  const deal = party?.data?.deal;
  return (
    <>
      <PageHeader
        back={deal ? { href: `/erp/sales/pipeline/${deal.id}`, label: deal.title } : { href: "/erp/sales/quotations", label: s.title }}
        eyebrow={s.title}
        title={s.newTitle}
        description={deal?.clientName ?? ss.doc.pickTitle}
      />
      <QuotationDealPicker deals={deals.data?.deals ?? []} dealId={dealId} />
      {party && !deal ? <ErpError message={party.error?.message} /> : null}
      {deal ? (
        <QuotationEditor
          quotation={null}
          dealId={deal.id}
          party={party.data}
          catalog={catalog?.data ?? []}
          retainerRules={retainer?.data ?? []}
        />
      ) : null}
    </>
  );
}
