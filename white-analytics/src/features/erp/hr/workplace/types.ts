import type { EmployeeRef } from "../types";

export type Asset = {
  id: string;
  assetCode: string;
  name: string;
  category: "laptop" | "phone" | "vehicle" | "furniture" | "other";
  serialNo: string;
  purchaseDate: string;
  purchaseValue: string;
  status: "available" | "assigned" | "repair" | "retired";
  holder?: EmployeeRef;
  notes: string;
};

export type AssetAssignment = {
  id: string;
  employee: EmployeeRef;
  assignedOn: string;
  returnedOn: string;
  conditionOut: string;
  conditionIn: string;
};

export type Training = {
  id: string;
  title: string;
  provider: string;
  startDate: string;
  endDate: string;
  location: string;
  cost: string;
  status: "planned" | "done" | "cancelled";
  notes: string;
  participantCount: number;
};

export type TrainingParticipant = {
  employee: EmployeeRef;
  status: "registered" | "attended" | "absent";
  score: string;
  certificate: string;
};

export type TimesheetEntry = {
  id: string;
  employee: EmployeeRef;
  workDate: string;
  projectId: string;
  projectName: string;
  hours: string;
  description: string;
  status: "draft" | "submitted" | "approved" | "rejected";
  rejectReason: string;
};

export type ProjectOption = { id: string; name: string; clientName: string };
