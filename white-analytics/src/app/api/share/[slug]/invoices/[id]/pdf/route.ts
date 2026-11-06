import { getShareContext } from "@/features/share/access";
import { proxyPortalFile } from "@/features/share/portal";

export const dynamic = "force-dynamic";

/** Invoice PDF for an unlocked share page; the services check the invoice belongs to this client. */
export async function GET(_req: Request, ctx: RouteContext<"/api/share/[slug]/invoices/[id]/pdf">) {
  const { slug, id } = await ctx.params;
  const share = await getShareContext(slug);
  if (share.status !== "ok") return Response.json({ error: "PIN_REQUIRED" }, { status: 401 });
  return proxyPortalFile(`/api/public/sales/portal/${encodeURIComponent(slug)}/invoices/${encodeURIComponent(id)}/pdf`, share.client.id);
}
