/** Browser calls through the gateway proxy. Errors come back as Indonesian messages for toasts. */
export async function proxyRequest<T>(url: string, init?: RequestInit): Promise<{ ok: true; data: T } | { ok: false; error: string }> {
  try {
    const res = await fetch(url, init);
    const body = await res.json().catch(() => ({}));
    if (!res.ok) return { ok: false, error: body.error ?? "Terjadi kesalahan." };
    return { ok: true, data: body as T };
  } catch {
    return { ok: false, error: "Layanan konten tidak dapat dihubungi." };
  }
}
