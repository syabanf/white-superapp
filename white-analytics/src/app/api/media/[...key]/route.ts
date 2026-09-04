import { getObject } from "@/lib/storage";

export const dynamic = "force-dynamic";

/**
 * Serves uploaded media from local storage (see src/lib/storage.ts).
 * Public by design: publishing providers (Meta Graph) fetch media by URL.
 * Keys are unguessable UUIDs, so exposure is equivalent to a signed URL.
 */
export async function GET(req: Request, ctx: RouteContext<"/api/media/[...key]">) {
  const { key } = await ctx.params;
  const obj = await getObject(key.join("/"));
  if (!obj) return new Response("Not found", { status: 404 });
  if (req.headers.get("if-none-match") === obj.etag) return new Response(null, { status: 304 });
  return new Response(new Uint8Array(obj.body), {
    headers: {
      "Content-Type": obj.contentType,
      "Content-Length": String(obj.body.byteLength),
      "Cache-Control": "public, max-age=31536000, immutable",
      ETag: obj.etag,
    },
  });
}
