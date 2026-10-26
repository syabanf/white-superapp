import type { Metadata } from "next";
import { PageHeader } from "@/components/dashboard/page-header";
import { requireErp } from "@/features/erp/access";
import { OfflineNotice } from "@/features/erp/offline-notice";
import { load } from "@/features/erp/shared/load";
import { ErpError } from "@/features/erp/shared/states";
import { EmployeeForm } from "@/features/erp/hr/components/employee-form";
import {
  listDepartments,
  listEmployeeOptions,
  listEmploymentStatuses,
  listPositions,
  listUserOptions,
} from "@/features/erp/hr/queries";
import { th } from "@/features/erp/hr/strings";

export const metadata: Metadata = { title: th.employees.newEmployee };

export default async function NewEmployeePage() {
  const { offline } = await requireErp("hr.write");
  const header = <PageHeader eyebrow={th.employees.title} title={th.employees.newEmployee} />;
  if (offline) return <>{header}<OfflineNotice message={offline.message} /></>;
  const [deps, pos, statuses, managers, users] = await Promise.all([
    load(() => listDepartments()),
    load(() => listPositions()),
    load(listEmploymentStatuses),
    load(listEmployeeOptions),
    load(listUserOptions),
  ]);
  if (!deps.data || !pos.data || !statuses.data)
    return <>{header}<ErpError message={(deps.error ?? pos.error ?? statuses.error)?.message} /></>;
  return (
    <>
      {header}
      <EmployeeForm
        departments={deps.data}
        positions={pos.data}
        statuses={statuses.data}
        managers={managers.data ?? []}
        users={users.data ?? []}
      />
    </>
  );
}
