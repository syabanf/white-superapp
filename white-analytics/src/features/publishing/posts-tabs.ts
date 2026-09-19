/**
 * Tabs of the "Semua post" list. Plain module (no "use client") because the
 * server page validates `?tab=` with `isPostsTab` and the client table renders them.
 */
import type { PostStatus } from "@/features/publishing/lib";
import type { PostStatusKey } from "@/lib/metrics/publishing";

export type PostsTab = "ALL" | "DRAFT" | "IN_REVIEW" | "APPROVED" | "SCHEDULED" | "PUBLISHED" | "FAILED";
export const POSTS_TABS: PostsTab[] = ["ALL", "DRAFT", "IN_REVIEW", "APPROVED", "SCHEDULED", "PUBLISHED", "FAILED"];

/** Which statuses each tab shows (REJECTED sits with drafts, PUBLISHING with scheduled). */
export const TAB_STATUSES: Record<PostsTab, PostStatus[] | null> = {
  ALL: null,
  DRAFT: ["DRAFT", "REJECTED"],
  IN_REVIEW: ["IN_REVIEW"],
  APPROVED: ["APPROVED"],
  SCHEDULED: ["SCHEDULED", "PUBLISHING"],
  PUBLISHED: ["PUBLISHED"],
  FAILED: ["FAILED"],
};

export function isPostsTab(v: string | undefined | null): v is PostsTab {
  return !!v && (POSTS_TABS as string[]).includes(v);
}

export function tabCount(tab: PostsTab, counts: Record<PostStatusKey, number>): number {
  const statuses = TAB_STATUSES[tab];
  if (!statuses) return Object.values(counts).reduce((a, b) => a + b, 0);
  return statuses.reduce((a, s) => a + counts[s], 0);
}
