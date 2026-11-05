import type { Metadata } from "next";
import { DocumentsListPage } from "@/features/erp/finance/components/documents-page";
import { fs } from "@/features/erp/finance/strings";

export const metadata: Metadata = { title: fs.documents.purchasingTitle };

export default async function PurchasingPage(props: PageProps<"/erp/finance/purchasing">) {
  return <DocumentsListPage kinds={["PR", "PO"]} title={fs.documents.purchasingTitle} subtitle={fs.documents.purchasingSubtitle} sp={await props.searchParams} />;
}
