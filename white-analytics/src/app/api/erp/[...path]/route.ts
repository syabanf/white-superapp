import type { NextRequest } from "next/server";
import { gatewaySession, gatewayUrl, GatewayError } from "@/lib/erp/gateway";

export const dynamic = "force-dynamic";

/**
 * Same-origin proxy for ERP client components: `/api/erp/v1/finance/...`
 * becomes `${GATEWAY_URL}/v1/finance/...` with the user's access token.
 * Bodies and responses stream through unchanged (JSON, multipart, PDF, XLSX).
 */
const FORWARD_REQUEST = ["content-type", "accept", "if-none-match"];
const FORWARD_RESPONSE = ["content-type", "content-disposition", "content-length", "etag", "cache-control"];

async function proxy(req: NextRequest, ctx: RouteContext<"/api/erp/[...path]">) {
  const { path } = await ctx.params;
  // Only the versioned API and file routes are reachable through here.
  if (path[0] !== "v1" && path[0] !== "api") {
    return Response.json(
      { ok: false, code: "NOT_FOUND", error: "Endpoint tidak ditemukan." },
      { status: 404 },
    );
  }
  let session;
  try {
    session = await gatewaySession();
  } catch (e) {
    const err =
      e instanceof GatewayError ? e : new GatewayError(503, "NETWORK", "Layanan ERP tidak dapat dihubungi.");
    return Response.json({ ok: false, code: err.code, error: err.message }, { status: err.status });
  }
  if (!session) {
    return Response.json(
      { ok: false, code: "UNAUTHENTICATED", error: "Silakan masuk terlebih dahulu." },
      { status: 401 },
    );
  }

  const headers = new Headers({ Authorization: `Bearer ${session.token}` });
  for (const h of FORWARD_REQUEST) {
    const v = req.headers.get(h);
    if (v) headers.set(h, v);
  }
  const forwardedFor = req.headers.get("x-forwarded-for");
  if (forwardedFor) headers.set("x-forwarded-for", forwardedFor);

  const target = `${gatewayUrl()}/${path.map(encodeURIComponent).join("/")}${req.nextUrl.search}`;
  const hasBody = req.method !== "GET" && req.method !== "HEAD";
  let upstream: Response;
  try {
    upstream = await fetch(target, {
      method: req.method,
      headers,
      body: hasBody ? req.body : undefined,
      // Required by undici when streaming a request body.
      ...(hasBody ? { duplex: "half" } : {}),
      cache: "no-store",
      redirect: "manual",
    } as RequestInit);
  } catch {
    return Response.json(
      { ok: false, code: "NETWORK", error: "Layanan ERP tidak dapat dihubungi. Pastikan backend berjalan." },
      { status: 503 },
    );
  }

  const out = new Headers();
  for (const h of FORWARD_RESPONSE) {
    const v = upstream.headers.get(h);
    if (v) out.set(h, v);
  }
  return new Response(upstream.body, { status: upstream.status, headers: out });
}

export const GET = proxy;
export const POST = proxy;
export const PUT = proxy;
export const PATCH = proxy;
export const DELETE = proxy;
