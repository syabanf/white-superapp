import "server-only";
import { gatewayFetch } from "@/lib/erp/gateway";
import type { OkrBoard } from "./types";

export async function getOkrBoard(period: string): Promise<OkrBoard> {
  const res = await gatewayFetch<Partial<OkrBoard>>(`/v1/performance/okr?period=${encodeURIComponent(period)}`);
  return {
    objectives: (res.objectives ?? []).map((o) => ({ ...o, keyResults: o.keyResults ?? [], progress: o.progress ?? 0 })),
    canCompany: res.canCompany ?? false,
    departments: res.departments ?? [],
    employees: res.employees ?? [],
  };
}
