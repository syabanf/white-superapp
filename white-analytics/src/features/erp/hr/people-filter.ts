import type { Department } from "./types";

type WithEmployee = { employee: { id: string; departmentName: string } };

/**
 * In-page `dept` (department id) and `employee` filters for HR lists whose
 * endpoint has no such filter; rows only carry the department name.
 */
export function filterPeople<T extends WithEmployee>(
  rows: T[],
  departments: Department[],
  f: { dept?: string; employee?: string },
): T[] {
  const deptName = f.dept ? departments.find((d) => d.id === f.dept)?.name : undefined;
  return rows.filter(
    (r) => (!f.dept || r.employee.departmentName === deptName) && (!f.employee || r.employee.id === f.employee),
  );
}
