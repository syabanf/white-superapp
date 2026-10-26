"use client";

import type { ColumnDef } from "@tanstack/react-table";
import { useRouter } from "next/navigation";
import { Link2 } from "lucide-react";
import { Avatar, AvatarFallback } from "@/components/ui/avatar";
import { ErpDataTable } from "@/features/erp/shared/data-table";
import { ErpStatusBadge } from "@/features/erp/shared/status-badge";
import { ResetFilters, SearchFilter, SelectFilter, type FilterOption } from "@/features/erp/shared/url-filters";
import { formatDate, initials } from "@/lib/format";
import { employeeStatusLabels, kinds, th } from "../strings";
import type { Employee } from "../types";
import { UrlSwitch } from "./ui";

export function EmployeesTable({ employees, departments }: { employees: Employee[]; departments: FilterOption[] }) {
  const router = useRouter();
  const e = th.employees;
  const columns: ColumnDef<Employee, unknown>[] = [
    {
      id: "name",
      accessorFn: (r) => r.fullName,
      header: e.name,
      cell: ({ row }) => (
        <span className="flex items-center gap-2.5">
          <Avatar className="size-8">
            <AvatarFallback className="text-xs font-semibold">{initials(row.original.fullName)}</AvatarFallback>
          </Avatar>
          <span className="min-w-0">
            <span className="flex items-center gap-1.5 truncate font-medium">
              {row.original.fullName}
              {row.original.userId ? <Link2 className="size-3 text-muted-foreground" aria-label={e.linkedUser} /> : null}
            </span>
            <span className="block truncate text-xs text-muted-foreground">{row.original.email || "–"}</span>
          </span>
        </span>
      ),
    },
    { accessorKey: "employeeNo", header: e.no, meta: { mono: true } },
    { accessorKey: "departmentName", header: th.department, cell: ({ getValue }) => getValue<string>() || "–" },
    { accessorKey: "positionName", header: th.position, cell: ({ getValue }) => getValue<string>() || "–" },
    {
      accessorKey: "joinDate",
      header: e.joinDate,
      cell: ({ getValue }) => <span className="tabular">{formatDate(getValue<string>())}</span>,
    },
    {
      accessorKey: "status",
      header: th.status,
      cell: ({ getValue }) => (
        <ErpStatusBadge status={getValue<string>()} labels={employeeStatusLabels} kinds={kinds.employee} />
      ),
    },
  ];

  return (
    <ErpDataTable
      columns={columns}
      data={employees}
      getRowId={(r) => r.id}
      onRowClick={(row) => router.push(`/erp/hr/employees/${row.original.id}`)}
      exportName="karyawan"
      toolbar={
        <>
          <SearchFilter placeholder={e.search} />
          <SelectFilter param="dept" options={departments} allLabel={th.allDepartments} label={th.department} />
          <UrlSwitch param="inactive" label={e.includeInactive} />
          <ResetFilters params={["q", "dept", "inactive"]} />
        </>
      }
    />
  );
}

