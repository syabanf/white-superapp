import { getShareContext } from "@/features/share/access";
import { proxyPortalFile } from "@/features/share/portal";

export const dynamic = "force-dynamic";

/** Design file of content waiting for the client's approval. */
export async function GET(_req: Request, ctx: RouteContext<"/api/share/[slug]/files/[id]">) {
  const { slug, id } = await ctx.params;
  const share = await getShareContext(slug);
  if (share.status !== "ok") return Response.json({ error: "PIN_REQUIRED" }, { status: 401 });
  return proxyPortalFile(`/api/public/content/portal/${encodeURIComponent(slug)}/files/${encodeURIComponent(id)}`, share.client.id);
}
