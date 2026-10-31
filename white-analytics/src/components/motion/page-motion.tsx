"use client";

import * as React from "react";
import { usePathname, useSearchParams } from "next/navigation";

/**
 * Replays the arrival animation (`.page-motion` in globals.css). The shared
 * layout never remounts, so the pathname key restarts it per page. A tab switch
 * keeps the header and tab bar mounted (the pill slides) and replays only the
 * sections below the tab bar. `display: contents` keeps the sections direct
 * flex items of the layout.
 */
export function PageMotion({ children }: { children: React.ReactNode }) {
  const pathname = usePathname();
  const tab = useSearchParams().get("tab");
  const ref = React.useRef<HTMLDivElement>(null);
  const lastTab = React.useRef(tab);

  React.useEffect(() => {
    if (lastTab.current === tab) return;
    lastTab.current = tab;
    const sections = [...(ref.current?.children ?? [])];
    const tabsAt = sections.findIndex((el) => el.matches('[data-slot="tabs"]') || el.querySelector('[data-slot="tabs"]'));
    for (const section of sections.slice(tabsAt + 1)) {
      for (const animation of section.getAnimations({ subtree: true })) {
        if (animation instanceof CSSAnimation && animation.animationName === "enter-up") {
          animation.cancel();
          animation.play();
        }
      }
    }
  }, [tab]);

  return (
    <div key={pathname} ref={ref} className="page-motion contents">
      {children}
    </div>
  );
}
