import { renderToBuffer } from "@react-pdf/renderer";
import { ReportDocument } from "./report-document";
import type { ReportPdfData } from "./types";

/** Render the report document to a PDF buffer (route handlers stay JSX-free). */
export async function renderReportPdf(data: ReportPdfData): Promise<Buffer> {
  return renderToBuffer(<ReportDocument data={data} />);
}
