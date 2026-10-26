"use client";

import type { HrDashboard } from "@/features/erp/hr/dashboard/types";
import { HrDrill } from "./drill";
import { AttendanceTab } from "./tab-attendance";
import { LeaveTab } from "./tab-leave";
import { PayrollTab } from "./tab-payroll";
import { PeopleTab } from "./tab-people";
import { SummaryTab } from "./tab-summary";

export type HrDashTab = "summary" | "attendance" | "leave" | "people" | "payroll";

/** Renders the selected dashboard tab from one payload. */
export function HrDashboardTab({ tab, data, leaveTypes }: { tab: HrDashTab; data: HrDashboard; leaveTypes: Record<string, string> }) {
  return (
    <>
      <TabBody tab={tab} data={data} leaveTypes={leaveTypes} />
      <HrDrill data={data} leaveTypes={leaveTypes} />
    </>
  );
}

function TabBody({ tab, data, leaveTypes }: { tab: HrDashTab; data: HrDashboard; leaveTypes: Record<string, string> }) {
  const props = { data, leaveTypes };
  switch (tab) {
    case "attendance":
      return <AttendanceTab {...props} />;
    case "leave":
      return <LeaveTab {...props} />;
    case "people":
      return <PeopleTab {...props} />;
    case "payroll":
      return <PayrollTab {...props} />;
    default:
      return <SummaryTab {...props} />;
  }
}
