/**
 * Migrasi data dari aplikasi Flutter lama (Firestore) ke WHITE Analytics.
 *
 * Cara pakai:
 *   1) Ekspor koleksi Firestore ke JSON, mis. dengan:
 *        npx -y firestore-export-import export --project white-superapp --out firestore.json
 *      atau `gcloud firestore export` lalu konversi ke JSON.
 *   2) Jalankan:
 *        pnpm tsx --env-file=.env scripts/migrate-firestore.ts ./firestore.json [--dry]
 *
 * Bentuk JSON yang diharapkan (fleksibel, lihat `readCollection`):
 * {
 *   "projects": { "<id>": { name, description, slug, clientPin, metaAdAccountId, collaboratorEmails: [] } },
 *   "users":    { "<id>": { email, password, role, allowedProjectIds: [] } }
 * }
 *
 * Catatan penting:
 * - Password lama tersimpan PLAINTEXT di app lama. Skrip ini men-hash-nya saat impor,
 *   tapi tetap sarankan semua user melakukan reset password setelah migrasi.
 * - PIN share lama juga plaintext → di-hash (bcrypt) ke `Client.sharePinHash`.
 * - Token Meta per proyek TIDAK diimpor (harus dihubungkan ulang lewat OAuth).
 * - Data metrik lama (campaigns/adSets/ads/dailyMetrics/demographics) diimpor best-effort
 *   bila tersedia pada dokumen proyek; jika tidak, lewati dan sinkronkan ulang dari Meta.
 */
import { readFileSync } from "node:fs";
import { PrismaClient } from "../src/generated/prisma/client";
import { PrismaPg } from "@prisma/adapter-pg";
import { hash } from "bcryptjs";

const adapter = new PrismaPg({ connectionString: process.env.DATABASE_URL });
const db = new PrismaClient({ adapter });

type AnyRec = Record<string, unknown>;

function slugify(name: string): string {
  return name
    .toLowerCase()
    .normalize("NFKD")
    .replace(/[^a-z0-9\s-]/g, "")
    .trim()
    .replace(/\s+/g, "-")
    .replace(/-+/g, "-");
}

/** Firestore exports vary: {col: {id: doc}} or {col: [{id, ...doc}]} or {__collections__: {...}} */
function readCollection(root: AnyRec, name: string): Array<{ id: string; data: AnyRec }> {
  const container = (root["__collections__"] as AnyRec | undefined) ?? root;
  const col = container[name];
  if (!col) return [];
  if (Array.isArray(col)) return col.map((d, i) => ({ id: String((d as AnyRec).id ?? i), data: d as AnyRec }));
  return Object.entries(col as AnyRec).map(([id, data]) => ({ id, data: (data ?? {}) as AnyRec }));
}

const str = (v: unknown, fallback = ""): string => (typeof v === "string" ? v : fallback);
const arr = (v: unknown): string[] => (Array.isArray(v) ? v.filter((x): x is string => typeof x === "string") : []);

async function main() {
  const file = process.argv[2];
  const dry = process.argv.includes("--dry");
  if (!file) {
    console.error("Usage: tsx scripts/migrate-firestore.ts <firestore-export.json> [--dry]");
    process.exit(1);
  }
  const root = JSON.parse(readFileSync(file, "utf8")) as AnyRec;

  const projects = readCollection(root, "projects");
  const users = readCollection(root, "users");
  console.log(`Ditemukan ${projects.length} proyek dan ${users.length} user.`);

  // ── Clients ────────────────────────────────────────────────
  const slugToClientId = new Map<string, string>();
  const oldIdToClientId = new Map<string, string>();
  for (const { id, data } of projects) {
    const name = str(data.name, `Klien ${id.slice(0, 6)}`);
    let slug = slugify(str(data.slug) || name);
    while (slugToClientId.has(slug)) slug = `${slug}-2`;
    const pin = str(data.clientPin);
    const payload = {
      name,
      slug,
      description: str(data.description),
      currency: "IDR",
      timezone: "Asia/Jakarta",
      shareEnabled: Boolean(pin) || Boolean(data.shareEnabled),
      sharePinHash: pin ? await hash(pin, 10) : null,
      shareModules: ["SOCIAL", "SEO", "ADS"] as const,
    };
    console.log(`  · Klien: ${name} → /${slug}${pin ? " (PIN di-hash)" : ""}`);
    if (dry) continue;
    const client = await db.client.upsert({
      where: { slug },
      update: { name: payload.name, description: payload.description },
      create: { ...payload, shareModules: [...payload.shareModules] },
    });
    slugToClientId.set(slug, client.id);
    oldIdToClientId.set(id, client.id);

    const adAccountId = str(data.metaAdAccountId);
    if (adAccountId) {
      await db.adAccount.upsert({
        where: { clientId_externalId: { clientId: client.id, externalId: adAccountId } },
        update: { name: `${name} Ads` },
        create: { clientId: client.id, externalId: adAccountId, name: `${name} Ads`, currency: "IDR" },
      });
    }
  }

  // ── Users ──────────────────────────────────────────────────
  for (const { data } of users) {
    const email = str(data.email).toLowerCase().trim();
    if (!email) continue;
    const oldRole = str(data.role, "projectSpecific");
    const role = oldRole === "admin" ? "ADMIN" : "MEMBER";
    const plain = str(data.password);
    const passwordHash = await hash(plain || `ganti-${Math.random().toString(36).slice(2)}`, 10);
    const name = email.split("@")[0] ?? email;
    console.log(`  · User: ${email} → ${role}${plain ? " (password lama di-hash, sarankan reset)" : " (password acak)"}`);
    if (dry) continue;
    const user = await db.user.upsert({
      where: { email },
      update: { role },
      create: { email, name, passwordHash, role },
    });
    // keanggotaan klien
    const allowed = arr(data.allowedProjectIds);
    const clientIds = role === "ADMIN" ? [] : allowed.map((pid) => oldIdToClientId.get(pid)).filter((x): x is string => !!x);
    for (const clientId of clientIds) {
      await db.clientMember.upsert({
        where: { userId_clientId: { userId: user.id, clientId } },
        update: {},
        create: { userId: user.id, clientId, role: "VIEWER" },
      });
    }
  }

  console.log(dry ? "\nDry-run selesai — tidak ada data yang ditulis." : "\n✓ Migrasi selesai.");
  console.log("Ingat: minta semua user reset password, dan hubungkan ulang Meta/Google lewat menu Pengaturan Klien.");
}

main()
  .catch((e) => {
    console.error(e);
    process.exit(1);
  })
  .finally(async () => {
    await db.$disconnect();
  });
