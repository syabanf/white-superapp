/**
 * Meta Graph API adapter — kontrak bersama untuk implementasi nyata & mock.
 * Semua tanggal keluar sebagai ISO `YYYY-MM-DD` / ISO timestamp (string) agar serializable.
 */

export type IgAccount = {
  id: string;
  username: string;
  name: string;
  biography: string | null;
  profilePictureUrl: string | null;
  followersCount: number;
  followsCount: number;
  mediaCount: number;
};

export type IgDailyInsight = {
  /** YYYY-MM-DD */
  date: string;
  reach: number;
  impressions: number;
  profileViews: number;
  /** net new followers hari itu (metric `follower_count`) */
  followerCount: number;
};

export type IgMediaType = "IMAGE" | "VIDEO" | "CAROUSEL_ALBUM";
export type IgMediaProductType = "FEED" | "REELS" | "STORY";

export type IgMedia = {
  id: string;
  caption: string;
  mediaType: IgMediaType;
  mediaProductType: IgMediaProductType | null;
  permalink: string | null;
  mediaUrl: string | null;
  thumbnailUrl: string | null;
  /** ISO timestamp terbit */
  timestamp: string;
  likeCount: number;
  commentsCount: number;
  /** insight per-media — hanya tersedia untuk akun sendiri */
  saved?: number;
  shares?: number;
  reach?: number;
  impressions?: number;
  views?: number;
};

/** Hasil Business Discovery untuk akun kompetitor (metrik publik saja). */
export type IgBusinessDiscovery = {
  username: string;
  name: string;
  biography: string | null;
  profilePictureUrl: string | null;
  followersCount: number;
  mediaCount: number;
  media: IgMedia[];
};

export type FbPage = {
  id: string;
  name: string;
  username: string | null;
  fanCount: number;
  followersCount: number;
};

export type FbPageDailyInsight = {
  /** YYYY-MM-DD */
  date: string;
  impressionsUnique: number;
  postEngagements: number;
};

export type FbPost = {
  id: string;
  message: string;
  /** ISO timestamp */
  createdTime: string;
  permalinkUrl: string | null;
  shares: number;
  reactions: number;
  comments: number;
};

export interface MetaGraphProvider {
  getInstagramAccount(token: string, igUserId: string): Promise<IgAccount>;
  getInstagramInsights(token: string, igUserId: string, since: Date, until: Date): Promise<IgDailyInsight[]>;
  getInstagramMedia(token: string, igUserId: string, limit?: number): Promise<IgMedia[]>;
  /** Business Discovery: profil + media publik kompetitor lewat akun IG sendiri. */
  discoverBusiness(token: string, igUserId: string, username: string): Promise<IgBusinessDiscovery>;
  getFacebookPage(token: string, pageId: string): Promise<FbPage>;
  getPageInsights(token: string, pageId: string, since: Date, until: Date): Promise<FbPageDailyInsight[]>;
  getPagePosts(token: string, pageId: string, limit?: number): Promise<FbPost[]>;
}
