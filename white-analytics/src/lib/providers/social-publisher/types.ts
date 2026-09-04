/**
 * Social publisher adapter — shared contract for the real Meta Graph
 * implementation and the deterministic mock.
 */

export type PublisherPlatform = "INSTAGRAM" | "FACEBOOK" | "TIKTOK";

export type PublishMedia = { url: string; kind: "IMAGE" | "VIDEO" };

export type PublishInput = {
  platform: PublisherPlatform;
  /** Decrypted page / IG user token. Ignored by the mock. */
  accessToken: string;
  /** IG user id / Facebook page id / TikTok open id */
  accountExternalId: string;
  /** Final caption (already includes per-target override + UTM link when relevant). */
  text: string;
  media: PublishMedia[];
  linkUrl?: string;
  /** Instagram: posted as the first comment right after publishing. */
  firstComment?: string;
};

export type PublishResult = {
  externalId: string;
  permalink: string | null;
};

export interface SocialPublisher {
  publish(input: PublishInput): Promise<PublishResult>;
}
