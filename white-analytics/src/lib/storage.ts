import "server-only";
import { createHash, randomUUID } from "node:crypto";
import { mkdir, readFile, stat, writeFile, unlink } from "node:fs/promises";
import path from "node:path";

/**
 * Object storage for uploaded media (publishing module).
 *
 * Local disk today (`UPLOAD_DIR`, default `<project>/.uploads`, gitignored),
 * served back through `GET /api/media/[...key]`. Keep every caller behind
 * `putObject` / `getObject` / `publicUrl` so swapping in S3/R2/Vercel Blob for
 * production is a one-file change. Keys are `<clientId>/<uuid>.<ext>`.
 */

export type StoredObject = { key: string; url: string; sizeBytes: number; contentType: string };

const ALLOWED: Record<string, string> = {
  "image/jpeg": "jpg",
  "image/png": "png",
  "image/webp": "webp",
  "image/gif": "gif",
  "video/mp4": "mp4",
  "video/quicktime": "mov",
};

export const MAX_UPLOAD_BYTES = 25 * 1024 * 1024;

function rootDir(): string {
  return process.env.UPLOAD_DIR || path.join(process.cwd(), ".uploads");
}

function safeKey(key: string): string {
  // Only `<segment>/<segment>` with a whitelisted charset — no traversal.
  if (!/^[A-Za-z0-9_-]+\/[A-Za-z0-9_.-]+$/.test(key)) throw new Error("INVALID_KEY");
  return key;
}

export function extensionFor(contentType: string): string | null {
  return ALLOWED[contentType] ?? null;
}

/**
 * Content type from the file's leading bytes. The browser-supplied MIME is a
 * claim, this is the evidence. Returns null for anything outside the allowlist.
 */
export function sniffContentType(data: Uint8Array): string | null {
  const b = data;
  const ascii = (from: number, to: number) => String.fromCharCode(...b.subarray(from, to));
  if (b.length >= 3 && b[0] === 0xff && b[1] === 0xd8 && b[2] === 0xff) return "image/jpeg";
  if (b.length >= 8 && b[0] === 0x89 && ascii(1, 4) === "PNG" && b[4] === 0x0d && b[5] === 0x0a && b[6] === 0x1a && b[7] === 0x0a) return "image/png";
  if (b.length >= 6 && (ascii(0, 6) === "GIF87a" || ascii(0, 6) === "GIF89a")) return "image/gif";
  if (b.length >= 12 && ascii(0, 4) === "RIFF" && ascii(8, 12) === "WEBP") return "image/webp";
  if (b.length >= 12 && ascii(4, 8) === "ftyp") return ascii(8, 12).startsWith("qt") ? "video/quicktime" : "video/mp4";
  return null;
}

export function publicUrl(key: string): string {
  const base = process.env.NEXT_PUBLIC_APP_URL?.replace(/\/$/, "") ?? "";
  return `${base}/api/media/${safeKey(key)}`;
}

export async function putObject(
  data: Uint8Array | Buffer,
  opts: { clientId: string; contentType: string },
): Promise<StoredObject> {
  const ext = extensionFor(opts.contentType);
  if (!ext || sniffContentType(data) !== opts.contentType) throw new Error("UNSUPPORTED_TYPE");
  if (data.byteLength > MAX_UPLOAD_BYTES) throw new Error("TOO_LARGE");
  const key = safeKey(`${opts.clientId}/${randomUUID()}.${ext}`);
  const abs = path.join(rootDir(), key);
  await mkdir(path.dirname(abs), { recursive: true });
  await writeFile(abs, data);
  return { key, url: publicUrl(key), sizeBytes: data.byteLength, contentType: opts.contentType };
}

export async function getObject(key: string): Promise<{ body: Buffer; contentType: string; etag: string } | null> {
  const k = safeKey(key);
  const abs = path.join(rootDir(), k);
  try {
    const [body, info] = await Promise.all([readFile(abs), stat(abs)]);
    const ext = k.split(".").pop() ?? "";
    const contentType = Object.entries(ALLOWED).find(([, e]) => e === ext)?.[0] ?? "application/octet-stream";
    const etag = createHash("sha1").update(`${info.size}-${info.mtimeMs}`).digest("hex");
    return { body, contentType, etag };
  } catch {
    return null;
  }
}

export async function deleteObject(key: string): Promise<void> {
  try {
    await unlink(path.join(rootDir(), safeKey(key)));
  } catch {
    /* already gone */
  }
}

/** Extract the storage key from a URL produced by `publicUrl`, else null (external URL). */
export function keyFromUrl(url: string): string | null {
  const m = url.match(/\/api\/media\/([A-Za-z0-9_-]+\/[A-Za-z0-9_.-]+)$/);
  return m ? m[1]! : null;
}
