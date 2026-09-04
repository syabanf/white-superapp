import { publishDuePosts } from "@/features/publishing/scheduler";

export const dynamic = "force-dynamic";

/**
 * Publish cron (every 5 min via vercel.json; Vercel Hobby only allows daily —
 * point an external scheduler here instead). Guarded by `Authorization: Bearer ${CRON_SECRET}`.
 * Claims due SCHEDULED posts atomically, so overlapping runs are safe.
 */
export async function GET(req: Request) {
  const secret = process.env.CRON_SECRET;
  if (!secret || req.headers.get("authorization") !== `Bearer ${secret}`) {
    return Response.json({ error: "UNAUTHORIZED" }, { status: 401 });
  }
  const startedAt = new Date();
  const result = await publishDuePosts({ now: startedAt });
  return Response.json({ ok: true, ranAt: startedAt.toISOString(), ...result });
}
