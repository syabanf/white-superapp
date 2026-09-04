import "server-only";
import { ProviderError } from "@/lib/action-result";
import { runActor } from "./client";
import { getApifyActors } from "./config";
import {
  normalizeFacebookPage,
  normalizeFacebookPost,
  normalizeInstagramPost,
  normalizeInstagramProfile,
  normalizeTikTokItems,
  profileUrl,
} from "./normalize";
import type { PublicPlatform, PublicPost, PublicProfile, PublicSocialProvider } from "./types";

const clean = (u: string) => u.replace(/^@/, "").trim().toLowerCase();

async function tiktok(username: string, limit: number) {
  const actors = getApifyActors();
  const items = await runActor(actors.tiktok, {
    profiles: [clean(username)],
    resultsPerPage: Math.min(Math.max(limit, 1), 100),
    profileSorting: "latest",
    profileScrapeSections: ["videos"],
    shouldDownloadVideos: false,
    shouldDownloadCovers: false,
    shouldDownloadSubtitles: false,
    shouldDownloadSlideshowImages: false,
  }, { timeoutSecs: 180, maxItems: limit });
  const { profile, posts } = normalizeTikTokItems(items, clean(username));
  if (!profile) throw new ProviderError("apify", "NOT_FOUND", `Akun TikTok @${clean(username)} tidak ditemukan atau belum punya video publik.`);
  return { profile, posts };
}

export const apifyPublicSocial: PublicSocialProvider = {
  async profile(platform: PublicPlatform, username: string): Promise<PublicProfile> {
    const actors = getApifyActors();
    const u = clean(username);
    if (platform === "INSTAGRAM") {
      const items = await runActor(actors.instagramProfile, { usernames: [u] }, { timeoutSecs: 120, maxItems: 1 });
      if (!items[0]) throw new ProviderError("apify", "NOT_FOUND", `Akun Instagram @${u} tidak ditemukan.`);
      return normalizeInstagramProfile(items[0], u);
    }
    if (platform === "TIKTOK") return (await tiktok(u, 1)).profile;
    const pages = await runActor(actors.facebookPage, { startUrls: [{ url: profileUrl("FACEBOOK", u) }] }, { timeoutSecs: 120, maxItems: 1 });
    if (!pages[0]) throw new ProviderError("apify", "NOT_FOUND", `Halaman Facebook ${u} tidak ditemukan.`);
    return normalizeFacebookPage(pages[0], u);
  },

  async posts(platform: PublicPlatform, username: string, limit = 30): Promise<PublicPost[]> {
    const actors = getApifyActors();
    const u = clean(username);
    if (platform === "INSTAGRAM") {
      const items = await runActor(actors.instagramPosts, {
        directUrls: [profileUrl("INSTAGRAM", u)],
        resultsType: "posts",
        resultsLimit: limit,
        addParentData: false,
      }, { timeoutSecs: 180, maxItems: limit });
      return items.map(normalizeInstagramPost).filter((p): p is PublicPost => p !== null);
    }
    if (platform === "TIKTOK") return (await tiktok(u, limit)).posts;
    const items = await runActor(actors.facebookPosts, { startUrls: [{ url: profileUrl("FACEBOOK", u) }], resultsLimit: limit }, { timeoutSecs: 180, maxItems: limit });
    return items.map(normalizeFacebookPost).filter((p): p is PublicPost => p !== null);
  },
};
