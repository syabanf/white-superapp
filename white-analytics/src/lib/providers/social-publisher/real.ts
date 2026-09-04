/**
 * Real publisher — Meta Graph API v21.0 with plain `fetch`.
 *
 * Instagram: create container(s) → `media_publish` (carousel via `children`,
 * video containers polled on `status_code`). Facebook Page: `/feed` for
 * text/link, `/photos` for one image, `/videos` for video. First comment via
 * `/{id}/comments`. TikTok throws PERMISSION until the Content Posting API
 * is approved. Graph errors are normalised through `mapMetaError`.
 */
import { ProviderError } from "@/lib/action-result";
import { mapMetaError } from "@/lib/providers/meta-graph/real";
import type { PublishInput, PublishMedia, PublishResult, SocialPublisher } from "./types";

const GRAPH_BASE = "https://graph.facebook.com/v21.0";
const VIDEO_POLL_MS = 3000;
const VIDEO_POLL_MAX = 40; // ~2 minutes

type GraphErrorBody = { error?: { message?: string; type?: string; code?: number; error_subcode?: number } };

async function graph<T>(method: "GET" | "POST", path: string, token: string, params: Record<string, string> = {}): Promise<T> {
  const url = new URL(`${GRAPH_BASE}/${path.replace(/^\//, "")}`);
  const body = new URLSearchParams({ ...params, access_token: token });
  if (method === "GET") for (const [k, v] of body) url.searchParams.set(k, v);
  let res: Response;
  try {
    res = await fetch(url.toString(), {
      method,
      cache: "no-store",
      body: method === "POST" ? body : undefined,
    });
  } catch (cause) {
    throw new ProviderError("meta", "NETWORK", "Tidak dapat menghubungi Meta Graph API", { cause });
  }
  const json = (await res.json().catch(() => null)) as (T & GraphErrorBody) | null;
  if (!res.ok || json == null || json.error) throw mapMetaError(res.status, json);
  return json;
}

const sleep = (ms: number) => new Promise((r) => setTimeout(r, ms));

async function waitForContainer(containerId: string, token: string): Promise<void> {
  for (let i = 0; i < VIDEO_POLL_MAX; i++) {
    const { status_code } = await graph<{ status_code?: string }>("GET", containerId, token, { fields: "status_code" });
    if (status_code === "FINISHED") return;
    if (status_code === "ERROR" || status_code === "EXPIRED") {
      throw new ProviderError("meta", "UNKNOWN", `Kontainer media Instagram gagal diproses (${status_code})`);
    }
    await sleep(VIDEO_POLL_MS);
  }
  throw new ProviderError("meta", "NETWORK", "Menunggu pemrosesan video Instagram terlalu lama");
}

async function igContainer(igUserId: string, token: string, media: PublishMedia, extra: Record<string, string>): Promise<string> {
  const params =
    media.kind === "VIDEO"
      ? { media_type: extra.is_carousel_item ? "VIDEO" : "REELS", video_url: media.url, ...extra }
      : { image_url: media.url, ...extra };
  const { id } = await graph<{ id: string }>("POST", `${igUserId}/media`, token, params);
  if (media.kind === "VIDEO") await waitForContainer(id, token);
  return id;
}

async function publishInstagram(input: PublishInput): Promise<PublishResult> {
  const { accessToken: token, accountExternalId: igUserId } = input;
  if (input.media.length === 0) throw new ProviderError("meta", "UNKNOWN", "Instagram membutuhkan minimal satu media");

  let creationId: string;
  if (input.media.length === 1) {
    creationId = await igContainer(igUserId, token, input.media[0]!, { caption: input.text });
  } else {
    const children: string[] = [];
    for (const m of input.media) children.push(await igContainer(igUserId, token, m, { is_carousel_item: "true" }));
    const { id } = await graph<{ id: string }>("POST", `${igUserId}/media`, token, {
      media_type: "CAROUSEL",
      children: children.join(","),
      caption: input.text,
    });
    creationId = id;
  }

  const { id: mediaId } = await graph<{ id: string }>("POST", `${igUserId}/media_publish`, token, { creation_id: creationId });
  const { permalink } = await graph<{ permalink?: string }>("GET", mediaId, token, { fields: "permalink" });
  if (input.firstComment?.trim()) {
    await graph("POST", `${mediaId}/comments`, token, { message: input.firstComment.trim() }).catch(() => undefined);
  }
  return { externalId: mediaId, permalink: permalink ?? null };
}

async function publishFacebook(input: PublishInput): Promise<PublishResult> {
  const { accessToken: token, accountExternalId: pageId } = input;
  const video = input.media.find((m) => m.kind === "VIDEO");
  const images = input.media.filter((m) => m.kind === "IMAGE");
  let postId: string;

  if (video) {
    const { id } = await graph<{ id: string }>("POST", `${pageId}/videos`, token, { file_url: video.url, description: input.text });
    postId = id;
  } else if (images.length === 1) {
    const { post_id, id } = await graph<{ id: string; post_id?: string }>("POST", `${pageId}/photos`, token, {
      url: images[0]!.url,
      message: input.text,
    });
    postId = post_id ?? id;
  } else if (images.length > 1) {
    // Multi-photo post: unpublished photos attached to one feed post.
    const attached: string[] = [];
    for (const img of images) {
      const { id } = await graph<{ id: string }>("POST", `${pageId}/photos`, token, { url: img.url, published: "false" });
      attached.push(id);
    }
    const params: Record<string, string> = { message: input.text };
    attached.forEach((id, i) => (params[`attached_media[${i}]`] = JSON.stringify({ media_fbid: id })));
    const { id } = await graph<{ id: string }>("POST", `${pageId}/feed`, token, params);
    postId = id;
  } else {
    const params: Record<string, string> = { message: input.text };
    if (input.linkUrl) params.link = input.linkUrl;
    const { id } = await graph<{ id: string }>("POST", `${pageId}/feed`, token, params);
    postId = id;
  }

  const { permalink_url } = await graph<{ permalink_url?: string }>("GET", postId, token, { fields: "permalink_url" }).catch(() => ({
    permalink_url: undefined,
  }));
  if (input.firstComment?.trim()) {
    await graph("POST", `${postId}/comments`, token, { message: input.firstComment.trim() }).catch(() => undefined);
  }
  return { externalId: postId, permalink: permalink_url ?? null };
}

export const realPublisher: SocialPublisher = {
  async publish(input) {
    if (input.platform === "INSTAGRAM") return publishInstagram(input);
    if (input.platform === "FACEBOOK") return publishFacebook(input);
    throw new ProviderError("tiktok", "PERMISSION", "Publikasi TikTok belum tersedia — menunggu persetujuan Content Posting API");
  },
};
