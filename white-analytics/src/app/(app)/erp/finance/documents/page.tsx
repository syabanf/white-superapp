import type { Metadata } from "next";
import { DocumentsListPage } from "@/features/erp/finance/components/documents-page";
import { fs } from "@/features/erp/finance/strings";

export const metadata: Metadata = { title: fs.documents.commercialTitle };

export default async function CommercialDocumentsPage(props: PageProps<"/erp/finance/documents">) {
  return <DocumentsListPage kinds={["QUOTATION", "INVOICE"]} title={fs.documents.commercialTitle} subtitle={fs.documents.commercialSubtitle} sp={await props.searchParams} />;
}
