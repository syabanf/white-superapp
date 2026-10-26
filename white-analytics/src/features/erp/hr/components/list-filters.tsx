"use client";

import { SearchSelectFilter, SelectFilter, type FilterOption } from "@/features/erp/shared/url-filters";
import type { EmployeeOption } from "./ui";
import { th } from "../strings";

/** Department (`dept`) and employee (`employee`) filters shared by the HR lists. */
export function PeopleFilters({ departments, employees }: { departments: FilterOption[]; employees?: EmployeeOption[] }) {
  return (
    <>
      {departments.length ? (
        <SelectFilter param="dept" options={departments} allLabel={th.allDepartments} label={th.department} />
      ) : null}
      {employees?.length ? (
        <SearchSelectFilter
          param="employee"
          placeholder={th.allEmployees}
          options={employees.map((e) => ({ value: e.id, label: e.label }))}
          className="w-56"
        />
      ) : null}
    </>
  );
}

export const PEOPLE_PARAMS = ["dept", "employee"];
