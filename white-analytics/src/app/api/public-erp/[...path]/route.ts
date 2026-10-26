import type { NextRequest } from "next/server";
import { gatewayUrl } from "@/lib/erp/gateway";

export const dynamic = "force-dynamic";

/**
 * Same-origin proxy for the public client pages (contract signing,
 * onboarding portal, lead forms, chat widget). Only `v1/public/...` and
 * `api/public/...` are reachable, and no token is ever attached.
 */
const FORWARD_REQUEST = ["content-type", "accept", "user-agent"];
const FORWARD_RESPONSE = ["content-type", "content-disposition", "content-length", "cache-control"];
const notFound = () => Response.json({ ok: false, code: "NOT_FOUND", error: "Endpoint tidak ditemukan." }, { status: 404 });

async function proxy(req: NextRequest, ctx: RouteContext<"/api/public-erp/[...path]">) {
  const { path } = await ctx.params;
  const isPublicPath = path[1] === "public" || (path[1] === "recruitment" && path[2] === "public");
  if ((path[0] !== "v1" && path[0] !== "api") || !isPublicPath || path.some((p) => p === ".." || p === ".")) return notFound();

  const headers = new Headers();
  for (const h of FORWARD_REQUEST) {
    const v = req.headers.get(h);
    if (v) headers.set(h, v);
  }
  const ip = req.headers.get("x-forwarded-for")?.split(",")[0]?.trim() || req.headers.get("x-real-ip");
  if (ip) headers.set("x-forwarded-for", ip);

  // Keep ":" literal so custom-method routes (":sign") still match on the gateway.
  const target = `${gatewayUrl()}/${path.map((p) => encodeURIComponent(p).replace(/%3A/gi, ":")).join("/")}${req.nextUrl.search}`;
  const hasBody = req.method !== "GET" && req.method !== "HEAD";
  let upstream: Response;
  try {
    upstream = await fetch(target, {
      method: req.method,
      headers,
      body: hasBody ? req.body : undefined,
      ...(hasBody ? { duplex: "half" } : {}),
      cache: "no-store",
      redirect: "manual",
    } as RequestInit);
  } catch {
    return Response.json(
      { ok: false, code: "NETWORK", error: "Layanan sedang tidak dapat dihubungi. Coba lagi beberapa saat." },
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
// psikotes answer autosave is a PUT
export const PUT = proxy;
