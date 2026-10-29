import "server-only";
import { db } from "@/lib/db";
import { runInDemoMode } from "@/lib/demo-mode";

export { inDemoMode, runInDemoMode } from "@/lib/demo-mode";

/** Klien bertanda demo? Klien yang tidak ditemukan dianggap bukan demo (pemanggil yang menolak). */
export async function isDemoClient(where: { id: string } | { slug: string }): Promise<boolean> {
  const c = await db.client.findUnique({ where, select: { isDemo: true } });
  return c?.isDemo === true;
}

/** `runInDemoMode` dengan flag diambil dari klien. */
export async function withClientMode<T>(where: { id: string } | { slug: string }, fn: () => Promise<T>): Promise<T> {
  return runInDemoMode(await isDemoClient(where), fn);
}
