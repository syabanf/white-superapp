"use client";

import { useCallback } from "react";
import { usePathname, useRouter, useSearchParams } from "next/navigation";

/** Sets or clears query parameters in place, so filtered lists stay linkable. */
export function useSetParams() {
  const router = useRouter();
  const pathname = usePathname();
  const sp = useSearchParams();
  return useCallback(
    (patch: Record<string, string | null>) => {
      const q = new URLSearchParams(sp.toString());
      for (const [key, value] of Object.entries(patch)) {
        if (value) q.set(key, value);
        else q.delete(key);
      }
      const s = q.toString();
      router.replace(s ? `${pathname}?${s}` : pathname, { scroll: false });
    },
    [pathname, router, sp],
  );
}

/** Sets or clears one query parameter in place. */
export function useSetParam() {
  const setParams = useSetParams();
  return useCallback((key: string, value: string) => setParams({ [key]: value }), [setParams]);
}
