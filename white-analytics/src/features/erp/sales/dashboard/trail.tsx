"use client";

import { DrillBreadcrumb, drillStrings, useDrillParams, type DrillCrumb } from "@/features/erp/shared/drill";

export type TrailLevel = { label: string; params: string[] };

/**
 * In-dashboard drill trail: "Semua > Maret 2026 > SEO" built from the
 * active filter levels in URL order of importance. Each crumb drops the
 * levels after it, so "Kembali" goes up one level.
 * Candidate for features/erp/shared/drill (performance and recruitment use it too).
 */
export function DashboardTrail({ levels, className }: { levels: TrailLevel[]; className?: string }) {
  const { hrefWith } = useDrillParams();
  if (levels.length === 0) return null;
  const drop = (from: number) => Object.fromEntries(levels.slice(from).flatMap((l) => l.params.map((p) => [p, null])));
  const items: DrillCrumb[] = [
    { label: drillStrings.all, href: hrefWith(drop(0)) },
    ...levels.map((l, i) => ({ label: l.label, href: i < levels.length - 1 ? hrefWith(drop(i + 1)) : undefined })),
  ];
  return <DrillBreadcrumb items={items} className={className} />;
}
