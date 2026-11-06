import "server-only";
import { cookies } from "next/headers";
import { GATEWAY_TIMEOUT, GatewayError, gatewayUrl } from "@/lib/erp/gateway";
import { publicFetch } from "@/lib/erp/public";

/**
 * The client portal (invoices, contracts, content approval) lives on the share
 * page but is served by the Go services, which check their own unlock token.
 * The PIN action stores that token in a second httpOnly cookie; it never
 * reaches the browser's JavaScript or a URL.
 */

const COOKIE_MAX_AGE = 12 * 60 * 60;

export function portalCookieName(clientId: string): string {
  return `share_portal_${clientId}`;
}

export async function getPortalToken(clientId: string): Promise<string | null> {
  const store = await cookies();
  return store.get(portalCookieName(clientId))?.value ?? null;
}

/** Trades the PIN for the services' unlock token. Best effort: the dashboard opens either way. */
export async function issuePortalToken(slug: string, pin: string, clientId: string): Promise<void> {
  try {
    const res = await fetch(`${gatewayUrl()}/v1/share/${encodeURIComponent(slug)}:verifyPin`, {
      method: "POST",
      headers: { "Content-Type": "application/json" },
      body: JSON.stringify({ pin }),
      cache: "no-store",
      signal: AbortSignal.timeout(GATEWAY_TIMEOUT.session),
    });
    const body = (await res.json().catch(() => ({}))) as { unlockToken?: string };
    if (!res.ok || !body.unlockToken) return;
    const store = await cookies();
    store.set(portalCookieName(clientId), body.unlockToken, {
      httpOnly: true,
      sameSite: "lax",
      secure: process.env.NODE_ENV === "production",
      maxAge: COOKIE_MAX_AGE,
      path: "/",
    });
  } catch {
    // Backend offline: the portal tabs ask for the PIN again later.
  }
}

export type PortalInvoice = {
  id: string;
  invoiceNumber: string;
  label: string;
  dealTitle: string;
  amount: string;
  paidTotal: string;
  dueDate: string;
  sentAt?: string;
};

export type PortalContract = {
  id: string;
  contractNumber: string;
  title: string;
  status: string;
  startDate: string;
  endDate: string;
  signedAt?: string;
  signToken: string;
};

export type PaymentProofStatus = "PENDING" | "APPROVED" | "REJECTED";

export type PortalProof = {
  id: string;
  invoiceId: string;
  invoiceNumber: string;
  amount: string;
  paidOn: string;
  payerName: string;
  fileName: string;
  status: PaymentProofStatus;
  rejectReason: string;
  createdAt: string;
};

export type PortalData = {
  clientName: string;
  bankLine: string;
  invoices: PortalInvoice[];
  contracts: PortalContract[];
  proofs: PortalProof[];
};

export type PortalContentItem = {
  id: string;
  title: string;
  plannedDate: string;
  platform: string;
  format: string;
  hook: string;
  caption: string;
  hashtags: string[];
  revisionRound: number;
  files: { id: string; fileName: string; mimeType: string }[];
};

export type PortalLoad = { status: "ok"; billing: PortalData; content: PortalContentItem[] } | { status: "locked" } | { status: "offline"; message: string };

/** Billing and content for an unlocked share page. "locked" means the services want the PIN again. */
export async function loadPortal(slug: string, clientId: string): Promise<PortalLoad> {
  const token = await getPortalToken(clientId);
  if (!token) return { status: "locked" };
  const body = JSON.stringify({ unlockToken: token });
  const at = encodeURIComponent(slug);
  try {
    const [billing, review] = await Promise.all([
      publicFetch<Partial<PortalData>>(`/v1/public/sales/portal/${at}:view`, { method: "POST", body }),
      publicFetch<{ items?: PortalContentItem[] }>(`/v1/public/content/portal/${at}:review`, { method: "POST", body }),
    ]);
    return {
      status: "ok",
      billing: {
        clientName: billing.clientName ?? "",
        bankLine: billing.bankLine ?? "",
        invoices: billing.invoices ?? [],
        contracts: billing.contracts ?? [],
        proofs: billing.proofs ?? [],
      },
      content: (review.items ?? []).map((it) => ({ ...it, hashtags: it.hashtags ?? [], files: it.files ?? [], revisionRound: it.revisionRound ?? 0 })),
    };
  } catch (e) {
    if (e instanceof GatewayError && (e.status === 404 || e.code === "NOT_FOUND")) return { status: "locked" };
    return { status: "offline", message: e instanceof GatewayError ? e.message : "Layanan sedang tidak dapat dihubungi." };
  }
}

/** Streams a portal file (invoice PDF, design file) from the gateway with the unlock token as a header. */
export async function proxyPortalFile(gatewayPath: string, clientId: string): Promise<Response> {
  const token = await getPortalToken(clientId);
  if (!token) return Response.json({ error: "PIN_REQUIRED" }, { status: 401 });
  let res: Response;
  try {
    res = await fetch(`${gatewayUrl()}${gatewayPath}`, {
      headers: { "X-Share-Token": token },
      cache: "no-store",
      signal: AbortSignal.timeout(GATEWAY_TIMEOUT.transfer),
    });
  } catch {
    return Response.json({ error: "OFFLINE" }, { status: 503 });
  }
  if (!res.ok || !res.body) return Response.json({ error: "NOT_FOUND" }, { status: res.status === 401 ? 401 : 404 });
  const headers = new Headers({ "Cache-Control": "private, no-store" });
  for (const h of ["content-type", "content-disposition", "content-length"]) {
    const v = res.headers.get(h);
    if (v) headers.set(h, v);
  }
  return new Response(res.body, { headers });
}
