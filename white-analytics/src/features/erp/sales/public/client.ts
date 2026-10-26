import { ps } from "../strings";

export class PublicError extends Error {
  constructor(
    public status: number,
    public code: string,
    message: string,
  ) {
    super(message);
  }
}

/** JSON call from public pages through the token-less proxy (`/api/public-erp` + path). */
export async function publicClientFetch<T>(path: string, init: RequestInit = {}): Promise<T> {
  let res: Response;
  try {
    res = await fetch(`/api/public-erp${path}`, { ...init, headers: { "Content-Type": "application/json", ...init.headers } });
  } catch {
    throw new PublicError(503, "NETWORK", ps.offline);
  }
  const body = await res.json().catch(() => ({}));
  if (!res.ok) throw new PublicError(res.status, body.code ?? "UNKNOWN", body.error ?? ps.offline);
  return body as T;
}

/** File → base64 without the data URL prefix (proto bytes field). */
export function fileToBase64(file: File): Promise<string> {
  return new Promise((resolve, reject) => {
    const reader = new FileReader();
    reader.onload = () => resolve(String(reader.result).split(",")[1] ?? "");
    reader.onerror = () => reject(reader.error);
    reader.readAsDataURL(file);
  });
}
