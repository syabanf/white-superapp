"use server";

import { revalidatePath } from "next/cache";
import { gatewayFetch } from "@/lib/erp/gateway";
import { runGateway } from "@/features/erp/shared/action";

const O = "/v1/performance/okr";

async function send(path: string, method: string, body?: unknown) {
  const res = await runGateway(() => gatewayFetch(path, { method, body: body === undefined ? undefined : JSON.stringify(body) }));
  if (res.ok) revalidatePath("/erp/performance/okr");
  return res;
}

export type ObjectiveForm = {
  id: string;
  period: string;
  level: string;
  departmentId: string;
  employeeId: string;
  parentId: string;
  title: string;
  description: string;
  status: string;
};

export async function saveObjectiveAction(input: ObjectiveForm) {
  return send(`${O}/objectives`, "POST", input);
}

export async function deleteObjectiveAction(id: string) {
  return send(`${O}/objectives/${encodeURIComponent(id)}`, "DELETE");
}

export type KeyResultForm = { id: string; objectiveId: string; title: string; metric: string; startValue: number; targetValue: number; unit: string };

export async function saveKeyResultAction(input: KeyResultForm) {
  return send(`${O}/key-results`, "POST", input);
}

export async function deleteKeyResultAction(id: string) {
  return send(`${O}/key-results/${encodeURIComponent(id)}`, "DELETE");
}

export async function checkInAction(id: string, input: { value: number; confidence: number; note: string }) {
  return send(`${O}/key-results/${encodeURIComponent(id)}:check-in`, "POST", input);
}
