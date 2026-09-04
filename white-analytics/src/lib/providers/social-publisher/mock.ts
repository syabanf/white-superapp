/**
 * Mock publisher — deterministic success with a fake permalink. A caption
 * containing `[fail]` throws a ProviderError so the retry flow can be
 * exercised end-to-end without credentials.
 */
import { ProviderError } from "@/lib/action-result";
import { hashSeed } from "@/lib/providers/meta-graph/mock";
import type { PublishInput, PublishResult, SocialPublisher } from "./types";

export const MOCK_FAIL_MARKER = "[fail]";

export function mockPermalink(platform: PublishInput["platform"], accountExternalId: string, externalId: string): string {
  if (platform === "INSTAGRAM") return `https://www.instagram.com/p/${externalId.slice(-11)}/`;
  if (platform === "FACEBOOK") return `https://www.facebook.com/${accountExternalId}/posts/${externalId}`;
  return `https://www.tiktok.com/@${accountExternalId}/video/${externalId}`;
}

export function mockPublish(input: PublishInput): PublishResult {
  if (input.text.includes(MOCK_FAIL_MARKER)) {
    throw new ProviderError("mock-publisher", "UNKNOWN", "Publikasi gagal (simulasi — caption mengandung [fail])");
  }
  const seed = hashSeed(`${input.platform}:${input.accountExternalId}:${input.text}:${input.media.map((m) => m.url).join(",")}`);
  const externalId = `mock_${input.platform.toLowerCase()}_${seed.toString(36)}${(seed % 997).toString(36)}`;
  return { externalId, permalink: mockPermalink(input.platform, input.accountExternalId, externalId) };
}

export const mockPublisher: SocialPublisher = {
  async publish(input) {
    return mockPublish(input);
  },
};
