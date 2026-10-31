import type { Metadata } from "next";
import Link from "next/link";
import { notFound } from "next/navigation";
import { Hourglass } from "lucide-react";
import { PageHeader } from "@/components/dashboard/page-header";
import { Alert, AlertDescription } from "@/components/ui/alert";
import { requireErp } from "@/features/erp/access";
import { OfflineNotice } from "@/features/erp/offline-notice";
import { load } from "@/features/erp/shared/load";
import { ErpError } from "@/features/erp/shared/states";
import { QuotationEditor } from "@/features/erp/sales/components/quotation-editor";
import { QuotationView } from "@/features/erp/sales/components/quotation-view";
import { QuotationActions } from "@/features/erp/sales/components/quotations";
import { SalesBadge } from "@/features/erp/sales/components/ui";
import { salesCaps } from "@/features/erp/sales/lib";
import {
  getDocumentParty,
  getQuotation,
  listCatalog,
  listQuotations,
  listRetainerDiscounts,
} from "@/features/erp/sales/queries";
import { APPROVAL_STATUS, QUOTATION_STATUS, ss } from "@/features/erp/sales/strings";
import { cn } from "@/lib/utils";

const s = ss.quotations;
export const metadata: Metadata = { title: s.title };

export default async function QuotationPage(props: PageProps<"/erp/sales/quotations/[id]">) {
  const { id } = await props.params;
  const { access, offline } = await requireErp("sales.read");
  if (offline) return <OfflineNotice message={offline.message} />;
  const caps = salesCaps(access);
  const res = await load(() => getQuotation(id));
  if (res.error?.code === "NOT_FOUND") notFound();
  if (!res.data) return <ErpError message={res.error?.message} />;
  const q = res.data;
  const editable = caps.write && q.status === "draft";
  const [catalog, retainer, siblings, party] = await Promise.all([
    editable ? load(() => listCatalog()) : null,
    editable ? load(listRetainerDiscounts) : null,
    load(() => listQuotations({ dealId: q.dealId, limit: 100 })),
    load(() => getDocumentParty(q.dealId)),
  ]);
  const versions = (siblings.data ?? [])
    .filter((x) => x.quoteNumber === q.quoteNumber)
    .sort((a, b) => a.version - b.version);

  return (
    <>
      <PageHeader
        back={{ href: `/erp/sales/pipeline/${q.dealId}`, label: q.dealTitle }}
        eyebrow={`${s.title} · ${s.version} ${q.version}`}
        title={q.quoteNumber}
        description={q.clientName}
        actions={<QuotationActions quotation={q} caps={caps} />}
      />
      <div className="flex flex-wrap items-center gap-2">
        <SalesBadge value={q.status} map={QUOTATION_STATUS} />
        {q.approvalStatus !== "none" ? <SalesBadge value={q.approvalStatus} map={APPROVAL_STATUS} /> : null}
        {s.generatedBy[q.generatedBy] ? (
          <span className="text-xs text-muted-foreground">{s.generatedBy[q.generatedBy]}</span>
        ) : null}
        {versions.length > 1 ? (
          <nav className="ml-auto flex items-center gap-1 text-xs" aria-label={s.versions}>
            <span className="text-muted-foreground">{s.versions}</span>
            {versions.map((v) => (
              <Link
                key={v.id}
                href={`/erp/sales/quotations/${v.id}`}
                className={cn(
                  "rounded-full border px-2 py-0.5",
                  v.id === q.id ? "border-brand bg-brand/10" : "hover:bg-accent",
                )}
              >
                v{v.version}
              </Link>
            ))}
          </nav>
        ) : null}
      </div>
      {q.approvalStatus === "pending" ? (
        <Alert className="rounded-xl">
          <Hourglass className="size-4" />
          <AlertDescription>{s.pendingApproval}</AlertDescription>
        </Alert>
      ) : null}
      {editable ? (
        <QuotationEditor
          quotation={q}
          dealId={q.dealId}
          party={party.data}
          catalog={catalog?.data ?? []}
          retainerRules={retainer?.data ?? []}
        />
      ) : (
        <>
          {caps.write && q.status !== "draft" ? (
            <p className="text-xs text-muted-foreground">{s.readOnly}</p>
          ) : null}
          <QuotationView q={q} party={party.data} />
        </>
      )}
    </>
  );
}
