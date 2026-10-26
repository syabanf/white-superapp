import type { ReactNode } from "react";
import { History } from "lucide-react";
import { Card, CardContent, CardHeader, CardTitle } from "@/components/ui/card";
import { ErpEmpty } from "@/features/erp/shared/states";
import { formatDate, formatDateTime } from "@/lib/format";
import { th } from "../strings";
import type { EmployeeProfile, EmploymentHistoryEntry } from "../types";

function Item({ label, value }: { label: string; value: ReactNode }) {
  return (
    <div className="flex min-w-0 flex-col gap-0.5">
      <dt className="text-xs text-muted-foreground">{label}</dt>
      <dd className="truncate text-sm">{value || "–"}</dd>
    </div>
  );
}

function Block({ title, children }: { title: string; children: ReactNode }) {
  return (
    <Card className="gap-3">
      <CardHeader>
        <CardTitle className="text-[15px]">{title}</CardTitle>
      </CardHeader>
      <CardContent>
        <dl className="grid grid-cols-1 gap-x-6 gap-y-3 sm:grid-cols-2">{children}</dl>
      </CardContent>
    </Card>
  );
}

/** Read-only employee record, grouped like the form. */
export function ProfileData({ p }: { p: EmployeeProfile }) {
  const f = th.employees.fields;
  const e = p.employee;
  return (
    <div className="grid gap-4 lg:grid-cols-2">
      <Block title={th.employees.sections.employment}>
        <Item label={f.employeeNo} value={e.employeeNo} />
        <Item label={f.employmentStatus} value={p.employmentStatusName} />
        <Item label={th.department} value={e.departmentName} />
        <Item label={th.position} value={e.positionName} />
        <Item label={th.employees.manager} value={p.managerName} />
        <Item label={th.employees.joinDate} value={formatDate(e.joinDate)} />
        <Item label={f.probationEndDate} value={p.probationEndDate ? formatDate(p.probationEndDate) : ""} />
        <Item label={f.endDate} value={e.endDate ? formatDate(e.endDate) : ""} />
        <Item label={th.employees.email} value={e.email} />
        <Item label={th.employees.phone} value={e.phone} />
      </Block>
      <Block title={th.employees.sections.personal}>
        <Item
          label={`${f.birthPlace}, ${f.birthDate.toLowerCase()}`}
          value={[p.birthPlace, p.birthDate ? formatDate(p.birthDate) : ""].filter(Boolean).join(", ")}
        />
        <Item label={f.gender} value={th.genders[p.gender]} />
        <Item label={f.maritalStatus} value={th.maritalStatuses[p.maritalStatus]} />
        <Item label={f.religion} value={p.religion} />
        <div className="sm:col-span-2">
          <Item label={f.address} value={<span className="whitespace-normal">{p.address}</span>} />
        </div>
      </Block>
      <Block title={th.employees.sections.identity}>
        <Item label={f.nik} value={p.nik} />
        <Item label={f.npwp} value={p.npwp} />
        <Item label={f.bpjsTkNo} value={p.bpjsTkNo} />
        <Item label={f.bpjsKesNo} value={p.bpjsKesNo} />
      </Block>
      <Block title={`${th.employees.sections.bank} · ${th.employees.sections.emergency}`}>
        <Item label={f.bankName} value={p.bankName} />
        <Item label={f.bankAccountNo} value={p.bankAccountNo} />
        <Item label={f.bankAccountName} value={p.bankAccountName} />
        <Item label={th.employees.linkedUser} value={p.userEmail || th.employees.notLinked} />
        <Item label={f.emergencyName} value={p.emergencyContactName} />
        <Item
          label={`${f.emergencyPhone} · ${f.emergencyRelation}`}
          value={[p.emergencyContactPhone, p.emergencyContactRelation].filter(Boolean).join(" · ")}
        />
      </Block>
    </div>
  );
}

function Change({ label, from, to }: { label: string; from: string; to: string }) {
  if (!from && !to) return null;
  if (from === to) return null;
  return (
    <p className="text-sm">
      <span className="text-muted-foreground">{label}: </span>
      {from || "–"} → <span className="font-medium">{to || "–"}</span>
    </p>
  );
}

export function HistoryList({ entries }: { entries: EmploymentHistoryEntry[] }) {
  if (!entries.length) return <ErpEmpty compact icon={<History className="size-5" />} title={th.history.empty} />;
  return (
    <ol className="relative flex flex-col gap-4 border-l pl-5">
      {entries.map((h) => (
        <li key={h.id} className="relative">
          <span className="absolute top-1.5 -left-[25px] size-2.5 rounded-full bg-brand" />
          <div className="flex flex-wrap items-baseline gap-2">
            <span className="text-sm font-medium">{th.history.changeType[h.changeType] ?? h.changeType}</span>
            <span className="text-xs text-muted-foreground">{formatDateTime(h.createdAt)}</span>
          </div>
          <Change label={th.history.status} from={h.prevEmploymentStatus} to={h.newEmploymentStatus} />
          <Change label={th.history.department} from={h.prevDepartment} to={h.newDepartment} />
          <Change label={th.history.position} from={h.prevPosition} to={h.newPosition} />
          {h.notes ? <p className="text-sm text-muted-foreground">{h.notes}</p> : null}
        </li>
      ))}
    </ol>
  );
}
