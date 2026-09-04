"use client";

import Link from "next/link";
import { usePathname, useSearchParams } from "next/navigation";
import { Tabs, TabsList, TabsTrigger } from "@/components/ui/tabs";
import { PlatformIcon, platformLabel } from "@/features/social/components/platform-icon";
import { PLATFORMS, type Platform } from "@/features/social/lib";
import { s } from "@/features/social/strings";

/**
 * Tab platform sebagai tautan (`?platform=`) — mempertahankan parameter
 * pencarian lain (rentang tanggal, compare). Default INSTAGRAM tanpa param.
 */
export function PlatformTabs({ current }: { current: Platform }) {
  const pathname = usePathname();
  const searchParams = useSearchParams();

  const hrefFor = (p: Platform) => {
    const q = new URLSearchParams(searchParams.toString());
    if (p === "INSTAGRAM") q.delete("platform");
    else q.set("platform", p);
    const qs = q.toString();
    return qs ? `${pathname}?${qs}` : pathname;
  };

  return (
    <Tabs value={current}>
      <TabsList aria-label={s.platformTabsAria}>
        {PLATFORMS.map((p) => (
          <TabsTrigger key={p} value={p} asChild>
            <Link href={hrefFor(p)} scroll={false}>
              <PlatformIcon platform={p} className="size-3.5" mono={current !== p} />
              {platformLabel(p)}
            </Link>
          </TabsTrigger>
        ))}
      </TabsList>
    </Tabs>
  );
}
