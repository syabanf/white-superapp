export type KeyResult = {
  id: string;
  objectiveId: string;
  title: string;
  metric: "number" | "percent" | "currency";
  startValue: number;
  targetValue: number;
  currentValue: number;
  unit: string;
  sortOrder: number;
  progress: number;
  confidence: number;
};

export type Objective = {
  id: string;
  period: string;
  level: "company" | "department" | "employee";
  departmentId: string;
  employeeId: string;
  ownerName: string;
  parentId: string;
  title: string;
  description: string;
  status: "active" | "done" | "cancelled";
  keyResults: KeyResult[];
  progress: number;
  canEdit: boolean;
};

export type OkrOwner = { id: string; name: string };

export type OkrBoard = { objectives: Objective[]; canCompany: boolean; departments: OkrOwner[]; employees: OkrOwner[] };
