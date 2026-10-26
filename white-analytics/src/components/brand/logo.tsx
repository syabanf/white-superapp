"use client";

import { useId } from "react";
import { cn } from "@/lib/utils";
import { LOGO_LOWER, LOGO_SOLID, LOGO_TEXT, LOGO_UPPER } from "./logo-paths";

/**
 * Mixist logo, vectorized from the official raster (shape overlap 99.3%). The
 * hexagon arcs carry their vertical gradients; navy and text colors come from
 * --logo-navy / --logo-text so the mark stays visible on dark surfaces.
 * Gradient ids are per instance: a duplicate id inside a hidden copy (the
 * closed mobile sidebar) would blank every other logo on the page.
 * `size` is the height of the mark; the full logo is about 3.9 times as wide.
 */

export function MixistLogo({
  className,
  variant = "full",
  size = 24,
}: {
  className?: string;
  variant?: "full" | "mark";
  size?: number;
}) {
  const id = useId().replace(/:/g, "");
  const full = variant === "full";
  const width = full ? 1933 : 436;
  return (
    <svg
      role="img"
      aria-label="Mixist"
      viewBox={`0 0 ${width} 493`}
      height={size}
      width={(size * width) / 493}
      className={cn("shrink-0", className)}
    >
      <defs>
        <linearGradient id={`${id}u`} gradientUnits="userSpaceOnUse" x1="0" y1="0" x2="0" y2="414">
          <stop offset="0" stopColor="var(--logo-navy)" />
          <stop offset=".4" stopColor="#7E256C" />
          <stop offset=".63" stopColor="#B62777" />
          <stop offset="1" stopColor="#EC247D" />
        </linearGradient>
        <linearGradient id={`${id}l`} gradientUnits="userSpaceOnUse" x1="0" y1="140" x2="0" y2="492">
          <stop offset="0" stopColor="#E9227C" />
          <stop offset=".35" stopColor="#A92672" />
          <stop offset=".49" stopColor="#8C286E" />
          <stop offset=".69" stopColor="#69256A" />
          <stop offset="1" stopColor="var(--logo-navy)" />
        </linearGradient>
      </defs>
      <path fill={`url(#${id}u)`} d={LOGO_UPPER} />
      <path fill={`url(#${id}l)`} d={LOGO_LOWER} />
      <path fill="#A01E63" d={LOGO_SOLID} />
      {full ? <path fill="var(--logo-text)" d={LOGO_TEXT} /> : null}
    </svg>
  );
}
