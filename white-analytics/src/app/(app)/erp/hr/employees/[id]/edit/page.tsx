import type { Metadata } from "next";
import { PageHeader } from "@/components/dashboard/page-header";
import { requireErp } from "@/features/erp/access";
import { OfflineNotice } from "@/features/erp/offline-notice";
import { load } from "@/features/erp/shared/load";
import { ErpError } from "@/features/erp/shared/states";
import { EmployeeForm } from "@/features/erp/hr/components/employee-form";
import {
  getEmployeeProfile,
  listDepartments,
  listEmployeeOptions,
  listEmploymentStatuses,
  listPositions,
} from "@/features/erp/hr/queries";
import { th } from "@/features/erp/hr/strings";

export const metadata: Metadata = { title: th.employees.editEmployee };

export default async function EditEmployeePage(props: PageProps<"/erp/hr/employees/[id]/edit">) {
  const { id } = await props.params;
  const { offline } = await requireErp("hr.write");
  if (offline) return <OfflineNotice message={offline.message} />;
  const [profile, deps, pos, statuses, managers] = await Promise.all([
    load(() => getEmployeeProfile(id)),
    load(() => listDepartments()),
    load(() => listPositions()),
    load(listEmploymentStatuses),
    load(listEmployeeOptions),
  ]);
  if (!profile.data || !deps.data || !pos.data || !statuses.data)
    return <ErpError message={(profile.error ?? deps.error ?? pos.error ?? statuses.error)?.message} />;
  return (
    <>
      <PageHeader eyebrow={th.employees.editEmployee} title={profile.data.employee.fullName} />
      <EmployeeForm
        profile={profile.data}
        departments={deps.data}
        positions={pos.data}
        statuses={statuses.data}
        managers={managers.data ?? []}
        users={[]}
      />
    </>
  );
}
