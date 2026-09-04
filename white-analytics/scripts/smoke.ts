/**
 * Smoke test: logs in with the seeded admin, then fetches every route and asserts
 * it returns 200 without a Next.js error page.
 *
 *   pnpm tsx --env-file=.env scripts/smoke.ts [baseUrl]
 *
 * Exit code 1 when any route fails.
 */
const BASE = process.argv[2] ?? process.env.SMOKE_BASE_URL ?? "http://localhost:3100";
const EMAIL = process.env.SMOKE_EMAIL ?? "admin@white.id";
const PASSWORD = process.env.SMOKE_PASSWORD ?? "admin12345";

const jar = new Map<string, string>();

function cookieHeader(): string {
  return [...jar.entries()].map(([k, v]) => `${k}=${v}`).join("; ");
}

function storeCookies(res: Response) {
  const raw = res.headers.getSetCookie?.() ?? [];
  for (const c of raw) {
    const [pair] = c.split(";");
    const idx = pair!.indexOf("=");
    if (idx > 0) jar.set(pair!.slice(0, idx).trim(), pair!.slice(idx + 1).trim());
  }
}

async function get(path: string, redirect: RequestRedirect = "manual"): Promise<Response> {
  const res = await fetch(`${BASE}${path}`, { headers: { cookie: cookieHeader() }, redirect });
  storeCookies(res);
  return res;
}

async function login(): Promise<void> {
  const csrfRes = await get("/api/auth/csrf", "follow");
  const { csrfToken } = (await csrfRes.json()) as { csrfToken: string };
  const body = new URLSearchParams({ email: EMAIL, password: PASSWORD, csrfToken, callbackUrl: `${BASE}/`, json: "true" });
  const res = await fetch(`${BASE}/api/auth/callback/credentials`, {
    method: "POST",
    headers: { "content-type": "application/x-www-form-urlencoded", cookie: cookieHeader() },
    body,
    redirect: "manual",
  });
  storeCookies(res);
  const hasSession = [...jar.keys()].some((k) => k.includes("session-token"));
  if (!hasSession) throw new Error(`Login gagal (status ${res.status}). Pastikan seed sudah dijalankan.`);
}

const ERROR_MARKERS = [
  "Application error: a server-side exception",
  "Unhandled Runtime Error",
  "This page could not be found",
  "__next_error__",
];

type Check = { path: string; expect?: number; contains?: string[] };

const CLIENT = "kopi-nusantara";
const ROUTES: Check[] = [
  { path: "/" },
  { path: "/clients" },
  { path: `/clients/${CLIENT}` },
  { path: `/clients/${CLIENT}?preset=7d` },
  { path: `/clients/${CLIENT}?preset=90d&compare=0` },
  { path: `/clients/${CLIENT}/social` },
  { path: `/clients/${CLIENT}/social?platform=FACEBOOK` },
  { path: `/clients/${CLIENT}/social?platform=TIKTOK` },
  { path: `/clients/${CLIENT}/social/competitors` },
  { path: `/clients/${CLIENT}/seo` },
  { path: `/clients/${CLIENT}/seo/keywords` },
  { path: `/clients/${CLIENT}/seo/keywords?view=striking` },
  { path: `/clients/${CLIENT}/seo/pages` },
  { path: `/clients/${CLIENT}/seo/audit` },
  { path: `/clients/${CLIENT}/ads` },
  { path: `/clients/${CLIENT}/reports` },
  { path: `/clients/${CLIENT}/reports?tab=history` },
  { path: `/clients/${CLIENT}/reports?tab=insight` },
  { path: `/clients/${CLIENT}/settings` },
  { path: "/admin/users" },
  { path: "/clients/adiharjo-property" },
  { path: "/clients/bali-villa-escapes" },
  { path: "/share/adiharjo-property" },
  { path: "/share/kopi-nusantara" },
];

async function main() {
  console.log(`Smoke test → ${BASE}\n`);

  // anonymous checks first (before the session cookie exists)
  const loginPage = await get("/login", "follow");
  const loginHtml = await loginPage.text();
  if (loginPage.status !== 200 || !loginHtml.includes("Masuk")) {
    console.log(`✗ /login (anonim) — status ${loginPage.status}`);
    process.exitCode = 1;
  } else {
    console.log("✓ /login (anonim)");
  }
  const guarded = await get("/clients", "manual");
  if (guarded.status !== 307 && guarded.status !== 302) {
    console.log(`✗ /clients tanpa sesi seharusnya redirect ke /login (status ${guarded.status})`);
    process.exitCode = 1;
  } else {
    console.log("✓ /clients tanpa sesi → redirect /login");
  }

  await login();
  console.log("✓ login\n");

  let failed = 0;
  for (const r of ROUTES) {
    let status = 0;
    let note = "";
    try {
      const res = await get(r.path, "follow");
      status = res.status;
      const html = await res.text();
      const marker = ERROR_MARKERS.find((m) => html.includes(m));
      const missing = (r.contains ?? []).filter((c) => !html.includes(c));
      if (status !== (r.expect ?? 200)) note = `status ${status}`;
      else if (marker) note = `error page: ${marker}`;
      else if (missing.length) note = `teks hilang: ${missing.join(", ")}`;
    } catch (e) {
      note = `exception: ${(e as Error).message}`;
    }
    if (note) {
      failed++;
      console.log(`✗ ${r.path}\n    ${note}`);
    } else {
      console.log(`✓ ${r.path} (${status})`);
    }
  }
  console.log(`\n${ROUTES.length - failed}/${ROUTES.length} rute OK`);
  if (failed) process.exit(1);
}

main().catch((e) => {
  console.error(e);
  process.exit(1);
});
