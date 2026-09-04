/**
 * Apify adapter — public-web data through Apify actors (no platform OAuth):
 * Google SERP positions for the rank tracker and public Instagram / TikTok /
 * Facebook profiles + posts for social analytics and competitors.
 * Dates are ISO strings so results stay serialisable.
 */
import type { SocialPlatform } from "@/generated/prisma/enums";

export type PublicPlatform = SocialPlatform; // INSTAGRAM | FACEBOOK | TIKTOK

export type PublicProfile = {
  platform: PublicPlatform;
  externalId: string;
  username: string;
  displayName: string;
  biography: string | null;
  avatarUrl: string | null;
  followers: number;
  following: number | null;
  mediaCount: number;
  /** total likes (TikTok "hearts") when the platform exposes it */
  totalLikes?: number | null;
};

export type PublicPost = {
  externalId: string;
  caption: string;
  mediaType: "IMAGE" | "VIDEO" | "CAROUSEL_ALBUM";
  productType: "FEED" | "REELS" | "STORY" | null;
  permalink: string | null;
  mediaUrl: string | null;
  thumbnailUrl: string | null;
  /** ISO timestamp */
  publishedAt: string;
  likes: number;
  comments: number;
  shares: number;
  saves: number;
  views: number;
};

export interface PublicSocialProvider {
  profile(platform: PublicPlatform, username: string): Promise<PublicProfile>;
  posts(platform: PublicPlatform, username: string, limit?: number): Promise<PublicPost[]>;
}

/** Actor ids (`owner/name`) per job — overridable from the setup wizard. */
export type ApifyActors = {
  serp: string;
  instagramProfile: string;
  instagramPosts: string;
  tiktok: string;
  facebookPage: string;
  facebookPosts: string;
};

export type ApifyUsage = {
  username: string | null;
  plan: string | null;
  monthlyUsageUsd: number | null;
  monthlyLimitUsd: number | null;
};
