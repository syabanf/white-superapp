import { db } from "@/lib/db";
import { previousRange, toISODate } from "@/lib/dates";
import { requireClientAccess } from "@/lib/rbac";
import { buildReportPdfData } from "@/features/reports/pdf/build-data";
import { renderReportPdf } from "@/features/reports/pdf/render";

export const dynamic = "force-dynamic";

/** Download a stored report as PDF. RBAC: session + client access (via the report's client slug). */
export async function GET(_req: Request, ctx: RouteContext<"/api/reports/[id]/pdf">) {
  const { id } = await ctx.params;
  const report = await db.report.findUnique({ where: { id }, include: { client: true } });
  if (!report) return Response.json({ error: "NOT_FOUND" }, { status: 404 });

  // Redirects to /login or an unauthorized page when the session/membership is missing.
  const { client } = await requireClientAccess(report.client.slug);

  const range = { from: report.dateFrom, to: report.dateTo };
  const previous = previousRange(range);
  const data = await buildReportPdfData({
    clientId: client.id,
    clientName: client.name,
    currency: client.currency,
    title: report.title,
    range,
    previous,
    compare: report.compare,
    modules: report.modules,
    language: report.language,
    aiSummary: report.aiSummary,
  });
  const buffer = await renderReportPdf(data);
  const filename = `Laporan_${client.slug}_${toISODate(range.from)}_${toISODate(range.to)}.pdf`;

  return new Response(new Uint8Array(buffer), {
    headers: {
      "Content-Type": "application/pdf",
      "Content-Disposition": `attachment; filename="${filename}"`,
      "Cache-Control": "no-store",
    },
  });
}
