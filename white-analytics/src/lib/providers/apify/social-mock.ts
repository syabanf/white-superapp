/**
 * Deterministic public-social mock (same shape as the Apify adapter) so the
 * competitor and sync flows behave identically with or without a token.
 */
import { hashSeed, mulberry32 } from "@/lib/providers/meta-graph/mock";
import { profileUrl } from "./normalize";
import type { PublicPlatform, PublicPost, PublicProfile, PublicSocialProvider } from "./types";

const CAPTIONS = [
  "Promo minggu ini — cek highlight kami ✨",
  "Terima kasih sudah mampir hari ini 🙏",
  "Behind the scenes tim kami ☕",
  "Tips singkat untuk kamu yang baru mulai",
  "Koleksi baru sudah tersedia!",
  "Giveaway! Baca syaratnya di caption",
  "Testimoni pelanggan minggu ini ❤️",
  "Jadwal buka libur panjang — simpan ya",
];

export const mockPublicSocial: PublicSocialProvider = {
  async profile(platform: PublicPlatform, username: string): Promise<PublicProfile> {
    const u = username.replace(/^@/, "").toLowerCase();
    const rand = mulberry32(hashSeed(`pub:${platform}:${u}`));
    const followers = Math.round(2_000 + rand() * 180_000);
    return {
      platform,
      externalId: `${platform.slice(0, 2).toLowerCase()}_pub_${u}`,
      username: u,
      displayName: u.replace(/[._]/g, " ").replace(/\b\w/g, (c) => c.toUpperCase()),
      biography: `Akun resmi ${u}. DM untuk kerja sama.`,
      avatarUrl: `https://api.dicebear.com/9.x/initials/svg?seed=${encodeURIComponent(u)}`,
      followers,
      following: platform === "FACEBOOK" ? null : Math.round(50 + rand() * 900),
      mediaCount: Math.round(40 + rand() * 600),
      totalLikes: platform === "TIKTOK" ? Math.round(followers * (3 + rand() * 12)) : null,
    };
  },

  async posts(platform: PublicPlatform, username: string, limit = 30): Promise<PublicPost[]> {
    const u = username.replace(/^@/, "").toLowerCase();
    const rand = mulberry32(hashSeed(`pubposts:${platform}:${u}`));
    const base = 2_000 + rand() * 180_000;
    const now = Date.now();
    return Array.from({ length: limit }, (_, i) => {
      const daysAgo = i * (1 + rand() * 2) + rand();
      const isVideo = platform === "TIKTOK" || rand() < 0.35;
      const er = 0.01 + rand() * 0.06;
      const likes = Math.round(base * er);
      return {
        externalId: `${platform.slice(0, 2).toLowerCase()}_${u}_${i}`,
        caption: CAPTIONS[i % CAPTIONS.length]!,
        mediaType: isVideo ? "VIDEO" : rand() < 0.3 ? "CAROUSEL_ALBUM" : "IMAGE",
        productType: isVideo ? "REELS" : "FEED",
        permalink: `${profileUrl(platform, u)}${platform === "TIKTOK" ? "/video/" : "p/"}${i + 1000}`,
        mediaUrl: null,
        thumbnailUrl: `https://picsum.photos/seed/${platform}-${u}-${i}/400/400`,
        publishedAt: new Date(now - daysAgo * 86_400_000).toISOString(),
        likes,
        comments: Math.round(likes * (0.02 + rand() * 0.06)),
        shares: Math.round(likes * (0.01 + rand() * 0.05)),
        saves: Math.round(likes * (0.02 + rand() * 0.08)),
        views: isVideo ? Math.round(likes * (8 + rand() * 30)) : 0,
      };
    });
  },
};
