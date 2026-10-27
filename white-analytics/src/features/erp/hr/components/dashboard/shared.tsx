"use client";

import { slotColor, MUTED_SERIES } from "@/components/dashboard/charts/chart-primitives";
import type { HrDashboard } from "@/features/erp/hr/dashboard/types";
import { thd } from "@/features/erp/hr/dashboard/strings";

export type TabProps = {
  data: HrDashboard;
  /** leave type code -> name */
  leaveTypes: Record<string, string>;
};

/** Department ids in the payload order with a fixed color each (color follows the entity). */
export function deptPalette(data: HrDashboard) {
  const list = data.departments.map((d, i) => ({
    id: d.departmentId,
    name: d.departmentId ? d.name : thd.noDepartment,
    color: d.departmentId && i < 8 ? slotColor(i) : MUTED_SERIES,
  }));
  return { list, name: (id: string) => list.find((d) => d.id === id)?.name ?? thd.noDepartment };
}

export const deptKey = (id: string) => `d_${id || "none"}`;
/** Inverse of deptKey; "" for employees without a department. */
export const deptFromKey = (key: string) => (key.startsWith("d_") && key !== "d_none" ? key.slice(2) : "");

export function Grid({ children, className = "" }: { children: React.ReactNode; className?: string }) {
  return <div className={`grid grid-cols-1 gap-(--gap) lg:grid-cols-3 ${className}`}>{children}</div>;
}
