"use client";

import { useCallback } from "react";
import { usePathname, useRouter, useSearchParams } from "next/navigation";

/** Sets or clears one query parameter in place, so filtered lists stay linkable. */
export function useSetParam() {
  const router = useRouter();
  const pathname = usePathname();
  const sp = useSearchParams();
  return useCallback(
    (key: string, value: string) => {
      const q = new URLSearchParams(sp.toString());
      if (value) q.set(key, value);
      else q.delete(key);
      const s = q.toString();
      router.replace(s ? `${pathname}?${s}` : pathname, { scroll: false });
    },
    [pathname, router, sp],
  );
}
