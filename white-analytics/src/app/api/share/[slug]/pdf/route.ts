import { getShareContext } from "@/features/share/access";
import { getLatestInsight } from "@/features/reports/queries";
import { buildReportPdfData } from "@/features/reports/pdf/build-data";
import { renderReportPdf } from "@/features/reports/pdf/render";
import { previousRange, rangeForPreset, toISODate, type PresetKey } from "@/lib/dates";

export const dynamic = "force-dynamic";

const SHARE_PRESETS: PresetKey[] = ["7d", "28d", "90d"];

/** Public share dashboard → PDF. Same PIN gate as the page (HMAC cookie). */
export async function GET(req: Request, ctx: RouteContext<"/api/share/[slug]/pdf">) {
  const { slug } = await ctx.params;
  const share = await getShareContext(slug);
  if (share.status === "not-found") return Response.json({ error: "NOT_FOUND" }, { status: 404 });
  if (share.status === "disabled") return Response.json({ error: "DISABLED" }, { status: 404 });
  if (share.status === "locked") return Response.json({ error: "PIN_REQUIRED" }, { status: 401 });

  const client = share.client;
  const url = new URL(req.url);
  const presetRaw = url.searchParams.get("preset");
  const preset: PresetKey = SHARE_PRESETS.includes(presetRaw as PresetKey) ? (presetRaw as PresetKey) : "28d";
  const range = rangeForPreset(preset);
  const previous = previousRange(range);

  const insight = await getLatestInsight(client.id, "OVERVIEW", range);
  const data = await buildReportPdfData({
    clientId: client.id,
    clientName: client.name,
    currency: client.currency,
    title: `Laporan Performa — ${client.name}`,
    range,
    previous,
    compare: true,
    modules: client.shareModules,
    language: "id",
    aiSummary: insight?.content ?? null,
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
