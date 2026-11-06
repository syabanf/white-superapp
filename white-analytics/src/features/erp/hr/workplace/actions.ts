"use server";

import { revalidatePath } from "next/cache";
import { gatewayFetch } from "@/lib/erp/gateway";
import { runGateway } from "@/features/erp/shared/action";
import type { Asset, AssetAssignment, TimesheetEntry, Training } from "./types";

const H = "/v1/hr";

async function send<T>(path: string, method: string, body?: unknown) {
  const res = await runGateway(() => gatewayFetch<T>(path, { method, body: body === undefined ? undefined : JSON.stringify(body) }));
  if (res.ok) {
    revalidatePath("/erp/hr", "layout");
    revalidatePath("/erp/me", "layout");
  }
  return res;
}

const id = (v: string) => encodeURIComponent(v);

export type AssetForm = {
  id: string;
  assetCode: string;
  name: string;
  category: string;
  serialNo: string;
  purchaseDate: string;
  purchaseValue: string;
  notes: string;
  status: string;
};

export async function saveAssetAction(input: AssetForm) {
  return send<Asset>(`${H}/assets`, "POST", input);
}

export async function assignAssetAction(assetId: string, input: { employeeId: string; assignedOn: string; condition: string }) {
  return send<Asset>(`${H}/assets/${id(assetId)}:assign`, "POST", input);
}

export async function returnAssetAction(assetId: string, input: { returnedOn: string; condition: string }) {
  return send<Asset>(`${H}/assets/${id(assetId)}:return`, "POST", input);
}

export type TrainingForm = {
  id: string;
  title: string;
  provider: string;
  startDate: string;
  endDate: string;
  location: string;
  cost: string;
  status: string;
  notes: string;
};

export async function saveTrainingAction(input: TrainingForm) {
  return send<Training>(`${H}/trainings`, "POST", input);
}

export async function deleteTrainingAction(trainingId: string) {
  return send(`${H}/trainings/${id(trainingId)}`, "DELETE");
}

export async function setTrainingParticipantsAction(trainingId: string, employeeIds: string[]) {
  return send(`${H}/trainings/${id(trainingId)}/participants`, "PUT", { employeeIds });
}

export async function recordTrainingResultAction(trainingId: string, employeeId: string, input: { status: string; score: string; certificate: string }) {
  return send(`${H}/trainings/${id(trainingId)}/participants/${id(employeeId)}:result`, "POST", input);
}

export type TimesheetForm = { id: string; workDate: string; projectId: string; projectName: string; hours: string; description: string };

export async function saveTimesheetEntryAction(input: TimesheetForm) {
  return send<TimesheetEntry>(`${H}/timesheet`, "POST", input);
}

export async function deleteTimesheetEntryAction(entryId: string) {
  return send(`${H}/timesheet/${id(entryId)}`, "DELETE");
}

export async function submitTimesheetAction(from: string, to: string) {
  return send<{ count?: number }>(`${H}/timesheet:submit`, "POST", { from, to });
}

export async function decideTimesheetAction(ids: string[], approve: boolean, reason = "") {
  return send<{ count?: number }>(`${H}/timesheet:decide`, "POST", { ids, approve, reason });
}

/** Lending history for the asset dialog (read only). */
export async function assetHistoryAction(assetId: string) {
  return runGateway(() => gatewayFetch<{ history?: AssetAssignment[] }>(`${H}/assets/${id(assetId)}`).then((r) => r.history ?? []));
}
