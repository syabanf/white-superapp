/**
 * Mock Meta Graph — data deterministik & masuk akal (seed dari string),
 * dipakai saat META_APP_ID/SECRET kosong. Tanpa jaringan, tanpa rahasia.
 */
import { addDays, eachDay, toISODate, todayUtc } from "@/lib/dates";
import type {
  FbPage,
  FbPageDailyInsight,
  FbPost,
  IgAccount,
  IgBusinessDiscovery,
  IgDailyInsight,
  IgMedia,
  MetaGraphProvider,
} from "./types";

/** FNV-1a 32-bit — seed numerik stabil dari string apa pun. */
export function hashSeed(input: string): number {
  let h = 2166136261 >>> 0;
  for (let i = 0; i < input.length; i++) {
    h ^= input.charCodeAt(i);
    h = Math.imul(h, 16777619) >>> 0;
  }
  return h >>> 0;
}

/** PRNG deterministik (sama dengan seed script). */
export function mulberry32(seed: number): () => number {
  let a = seed >>> 0;
  return () => {
    a = (a + 0x6d2b79f5) >>> 0;
    let t = a;
    t = Math.imul(t ^ (t >>> 15), t | 1);
    t ^= t + Math.imul(t ^ (t >>> 7), t | 61);
    return ((t ^ (t >>> 14)) >>> 0) / 4294967296;
  };
}

/**
 * Deret followers harian yang plausible, panjang `days`, berakhir TEPAT di
 * `endFollowers`. Deterministik per `seedKey` — dipakai untuk membuat riwayat
 * snapshot kompetitor baru.
 */
export function buildFollowerSeries(seedKey: string, days: number, endFollowers: number): number[] {
  const rand = mulberry32(hashSeed(`series:${seedKey}`));
  const gain = Math.max(1, Math.round(endFollowers * (0.0004 + rand() * 0.0008)));
  let f = Math.max(10, endFollowers - gain * days);
  const out: number[] = [];
  for (let i = 0; i < days; i++) {
    const spike = rand() < 0.02 ? 4 + rand() * 3 : 1;
    f = Math.max(0, Math.round(f + gain * (0.7 + rand() * 0.6) * spike + (rand() - 0.5) * gain));
    out.push(f);
  }
  const diff = endFollowers - out[out.length - 1]!;
  return out.map((v, i) => Math.max(0, v + Math.round((diff * (i + 1)) / days)));
}

const CAPTIONS = [
  "Koleksi terbaru sudah tersedia! Cek link di bio ✨",
  "Terima kasih untuk semua pelanggan setia bulan ini 🙏",
  "Promo spesial akhir pekan — jangan sampai kehabisan!",
  "Behind the scenes dari tim kami hari ini 📸",
  "Tips singkat untuk kamu yang baru mulai. Simpan dulu!",
  "Giveaway time! Tag 3 temanmu di kolom komentar 🎁",
  "Produk favorit pelanggan minggu ini — sudah coba?",
  "Cerita di balik layar. Geser untuk lihat prosesnya →",
];
const HASHTAGS = "#promo #indonesia #lokal #umkm";

function titleCase(username: string): string {
  return (
    username
      .split(/[._]/)
      .filter(Boolean)
      .map((w) => w[0]!.toUpperCase() + w.slice(1))
      .join(" ") || username
  );
}

function baseFollowers(seedKey: string, min = 8_000, span = 290_000): number {
  return min + (hashSeed(seedKey) % span);
}

/** Media deterministik: tersebar ±90 hari ke belakang, ER wajar per tipe. */
function mockIgMedia(seedKey: string, count: number, followers: number, withInsights: boolean): IgMedia[] {
  const rand = mulberry32(hashSeed(`media:${seedKey}`));
  const end = todayUtc();
  const out: IgMedia[] = [];
  for (let i = 0; i < count; i++) {
    const daysAgo = Math.min(115, Math.round(i * (90 / Math.max(1, count)) + rand() * 3));
    const hourWib = 8 + Math.floor(rand() * 13); // 08–20 WIB
    const ts = new Date(addDays(end, -daysAgo).getTime() + (hourWib - 7) * 3_600_000 + Math.floor(rand() * 60) * 60_000);
    const roll = rand();
    const mediaType = roll < 0.42 ? "VIDEO" : roll < 0.72 ? "CAROUSEL_ALBUM" : ("IMAGE" as const);
    const er = (mediaType === "VIDEO" ? 0.03 : mediaType === "CAROUSEL_ALBUM" ? 0.024 : 0.018) * (0.5 + rand());
    const engagements = Math.max(8, Math.round(followers * er));
    const likeCount = Math.round(engagements * (0.82 + rand() * 0.08));
    const commentsCount = Math.max(1, Math.round(engagements * (0.05 + rand() * 0.04)));
    const media: IgMedia = {
      id: `mock_${hashSeed(`${seedKey}:${i}`).toString(36)}`,
      caption: `${CAPTIONS[(hashSeed(seedKey) + i) % CAPTIONS.length]} ${HASHTAGS}`,
      mediaType,
      mediaProductType: mediaType === "VIDEO" ? "REELS" : "FEED",
      permalink: `https://www.instagram.com/p/mock${hashSeed(`${seedKey}:${i}`).toString(36)}/`,
      mediaUrl: `https://picsum.photos/seed/mg-${hashSeed(seedKey) % 9973}-${i}/800/800`,
      thumbnailUrl: `https://picsum.photos/seed/mg-${hashSeed(seedKey) % 9973}-${i}/400/400`,
      timestamp: ts.toISOString(),
      likeCount,
      commentsCount,
    };
    if (withInsights) {
      const shares = Math.round(engagements * (0.04 + rand() * 0.05));
      const saved = Math.max(0, engagements - likeCount - commentsCount - shares);
      const reach = Math.round(followers * (mediaType === "VIDEO" ? 0.4 : 0.22) * (0.6 + rand() * 0.9));
      media.shares = shares;
      media.saved = saved;
      media.reach = reach;
      media.impressions = Math.round(reach * (1.3 + rand() * 0.5));
      media.views = mediaType === "VIDEO" ? Math.round(reach * (1.4 + rand())) : 0;
    }
    out.push(media);
  }
  // API mengembalikan terbaru dulu
  return out.sort((a, b) => (a.timestamp < b.timestamp ? 1 : -1));
}

export const mockMetaGraph: MetaGraphProvider = {
  async getInstagramAccount(_token, igUserId): Promise<IgAccount> {
    const key = `ig:${igUserId}`;
    const followersCount = baseFollowers(key, 12_000, 180_000);
    const username = `akun_${(hashSeed(key) % 46_655).toString(36)}`;
    return {
      id: igUserId,
      username,
      name: titleCase(username),
      biography: "Akun demo Instagram Business.",
      profilePictureUrl: `https://api.dicebear.com/9.x/initials/svg?seed=${encodeURIComponent(username)}&backgroundColor=1c1c1e&textColor=ffffff`,
      followersCount,
      followsCount: 180 + (hashSeed(key) % 240),
      mediaCount: 150 + (hashSeed(key) % 900),
    };
  },

  async getInstagramInsights(_token, igUserId, since, until): Promise<IgDailyInsight[]> {
    const followers = baseFollowers(`ig:${igUserId}`, 12_000, 180_000);
    return eachDay({ from: since, to: until }).map((d) => {
      const r = mulberry32(hashSeed(`ins:${igUserId}:${toISODate(d)}`));
      const reach = Math.round(followers * (0.08 + r() * 0.06));
      return {
        date: toISODate(d),
        reach,
        impressions: Math.round(reach * (1.4 + r() * 0.5)),
        profileViews: Math.round(reach * (0.05 + r() * 0.04)),
        followerCount: Math.round(followers * 0.0008 * (0.4 + r())),
      };
    });
  },

  async getInstagramMedia(_token, igUserId, limit = 30): Promise<IgMedia[]> {
    const followers = baseFollowers(`ig:${igUserId}`, 12_000, 180_000);
    return mockIgMedia(`ig:${igUserId}`, limit, followers, true);
  },

  async discoverBusiness(_token, _igUserId, username): Promise<IgBusinessDiscovery> {
    const u = username.trim().replace(/^@+/, "").toLowerCase();
    const key = `bd:${u}`;
    const followersCount = baseFollowers(key);
    const name = titleCase(u);
    return {
      username: u,
      name,
      biography: `Akun resmi ${name}. Demo Business Discovery.`,
      profilePictureUrl: `https://api.dicebear.com/9.x/initials/svg?seed=${encodeURIComponent(name)}&backgroundColor=8e8e93&textColor=ffffff`,
      followersCount,
      mediaCount: 150 + (hashSeed(key) % 1400),
      media: mockIgMedia(key, 25, followersCount, false),
    };
  },

  async getFacebookPage(_token, pageId): Promise<FbPage> {
    const key = `fb:${pageId}`;
    const followersCount = baseFollowers(key, 5_000, 120_000);
    const username = `page_${(hashSeed(key) % 46_655).toString(36)}`;
    return {
      id: pageId,
      name: titleCase(username),
      username,
      fanCount: Math.round(followersCount * 0.94),
      followersCount,
    };
  },

  async getPageInsights(_token, pageId, since, until): Promise<FbPageDailyInsight[]> {
    const followers = baseFollowers(`fb:${pageId}`, 5_000, 120_000);
    return eachDay({ from: since, to: until }).map((d) => {
      const r = mulberry32(hashSeed(`fbi:${pageId}:${toISODate(d)}`));
      const impressionsUnique = Math.round(followers * (0.05 + r() * 0.05));
      return {
        date: toISODate(d),
        impressionsUnique,
        postEngagements: Math.round(impressionsUnique * (0.03 + r() * 0.03)),
      };
    });
  },

  async getPagePosts(_token, pageId, limit = 25): Promise<FbPost[]> {
    const key = `fb:${pageId}`;
    const followers = baseFollowers(key, 5_000, 120_000);
    return mockIgMedia(key, limit, followers, false).map((m, i) => ({
      id: `${pageId}_post_${i}`,
      message: m.caption,
      createdTime: m.timestamp,
      permalinkUrl: `https://www.facebook.com/${pageId}/posts/${hashSeed(`${key}:${i}`)}`,
      shares: Math.round(m.likeCount * 0.05),
      reactions: m.likeCount,
      comments: m.commentsCount,
    }));
  },
};
