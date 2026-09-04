/**
 * Pure helpers khusus modul Social (client-safe, tanpa akses server).
 * Rumus umum tetap di `@/lib/metrics` — di sini hanya komposisi/utilitas modul.
 */
import { avgEngagementRate, contentType, type PostLike } from "@/lib/metrics";
import { dayKey, eachDay, type DateRange } from "@/lib/dates";

export const PLATFORMS = ["INSTAGRAM", "FACEBOOK", "TIKTOK"] as const;
export type Platform = (typeof PLATFORMS)[number];

/** Baca `?platform=` dari searchParams; default INSTAGRAM. */
export function parsePlatform(raw: string | string[] | undefined): Platform {
  const v = (Array.isArray(raw) ? raw[0] : raw)?.toUpperCase();
  return v === "FACEBOOK" || v === "TIKTOK" ? v : "INSTAGRAM";
}

export type SnapshotLike = {
  date: Date;
  followers: number;
  reach?: number | null;
  impressions?: number | null;
};

/** Followers pada tanggal tertentu = snapshot terakhir pada/sebelum tanggal itu (snaps terurut naik). */
export function followersAt(snaps: SnapshotLike[], date: Date): number {
  let f = 0;
  for (const sn of snaps) {
    if (sn.date > date) break;
    f = sn.followers;
  }
  return f;
}

export type DailySocialPoint = {
  date: string;
  followers: number | null;
  reach: number;
  impressions: number;
};

/**
 * Seri harian dari snapshot: followers di-forward-fill (hari tanpa snapshot memakai
 * nilai terakhir yang diketahui), reach/impresi dijumlah per hari (0 bila kosong).
 */
export function dailyFollowerSeries(range: DateRange, snaps: SnapshotLike[]): DailySocialPoint[] {
  const byDay = new Map<string, SnapshotLike>();
  let carry: number | null = null;
  for (const sn of snaps) {
    if (sn.date < range.from) carry = sn.followers;
    else byDay.set(dayKey(sn.date), sn);
  }
  return eachDay(range).map((d) => {
    const sn = byDay.get(dayKey(d));
    if (sn) carry = sn.followers;
    return {
      date: dayKey(d),
      followers: sn ? sn.followers : carry,
      reach: sn?.reach ?? 0,
      impressions: sn?.impressions ?? 0,
    };
  });
}

/**
 * Indeks seri ke 100 pada nilai pertama yang > 0 (aturan dataviz: dua ukuran beda
 * skala pada satu sumbu → indeks, bukan dual-axis). Nilai kosong/nol → null.
 */
export function indexTo100(values: (number | null)[]): (number | null)[] {
  const base = values.find((v): v is number => v != null && v > 0);
  if (base == null) return values.map(() => null);
  return values.map((v) => (v == null || v <= 0 ? null : (v / base) * 100));
}

export type ErByType = { type: string; avgEr: number; count: number };

/** Rata-rata ER per tipe konten (Reels/Video/Carousel/Foto/Story), urut ER menurun. */
export function erByContentType(posts: PostLike[], followers: number): ErByType[] {
  const groups = new Map<string, PostLike[]>();
  for (const p of posts) {
    const key = contentType(p);
    const list = groups.get(key) ?? [];
    list.push(p);
    groups.set(key, list);
  }
  return [...groups.entries()]
    .map(([type, list]) => ({ type, avgEr: avgEngagementRate(list, followers), count: list.length }))
    .sort((a, b) => b.avgEr - a.avgEr);
}

/**
 * Normalisasi username Instagram: buang `@` di depan, lowercase, trim.
 * Valid: 2–30 karakter [a-z0-9._], tidak diawali/diakhiri titik. Null bila tidak valid.
 */
export function normalizeIgUsername(raw: string): string | null {
  const u = raw.trim().replace(/^@+/, "").toLowerCase();
  if (u.length < 2 || u.length > 30) return null;
  if (!/^[a-z0-9._]+$/.test(u)) return null;
  if (u.startsWith(".") || u.endsWith(".")) return null;
  return u;
}
