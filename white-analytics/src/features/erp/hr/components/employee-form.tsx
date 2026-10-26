"use client";

import * as React from "react";
import { useRouter } from "next/navigation";
import { Loader2 } from "lucide-react";
import { Button } from "@/components/ui/button";
import { Card, CardContent, CardHeader, CardTitle } from "@/components/ui/card";
import { Input } from "@/components/ui/input";
import { Textarea } from "@/components/ui/textarea";
import { useErpAction } from "@/features/erp/shared/client";
import { DatePicker } from "@/features/erp/shared/period-picker";
import { saveEmployeeAction } from "../actions";
import { th } from "../strings";
import type { Department, EmployeeInput, EmployeeProfile, EmploymentStatusOption, Position, UserOption } from "../types";
import { EmployeePicker, Fld, SimpleSelect, useFieldErrors, type EmployeeOption } from "./ui";

const f = th.employees.fields;

function fromProfile(p?: EmployeeProfile): EmployeeInput {
  const e = p?.employee;
  return {
    employeeNo: e?.employeeNo ?? "",
    fullName: e?.fullName ?? "",
    email: e?.email ?? "",
    phone: e?.phone ?? "",
    departmentId: e?.departmentId ?? "",
    positionId: e?.positionId ?? "",
    managerId: e?.managerId ?? "",
    employmentStatus: p?.employmentStatus || "probation",
    joinDate: e?.joinDate ?? "",
    endDate: e?.endDate ?? "",
    probationEndDate: p?.probationEndDate ?? "",
    nik: p?.nik ?? "",
    npwp: p?.npwp ?? "",
    bpjsTkNo: p?.bpjsTkNo ?? "",
    bpjsKesNo: p?.bpjsKesNo ?? "",
    birthPlace: p?.birthPlace ?? "",
    birthDate: p?.birthDate ?? "",
    gender: p?.gender ?? "",
    maritalStatus: p?.maritalStatus ?? "",
    religion: p?.religion ?? "",
    address: p?.address ?? "",
    bankName: p?.bankName ?? "",
    bankAccountNo: p?.bankAccountNo ?? "",
    bankAccountName: p?.bankAccountName ?? "",
    emergencyContactName: p?.emergencyContactName ?? "",
    emergencyContactPhone: p?.emergencyContactPhone ?? "",
    emergencyContactRelation: p?.emergencyContactRelation ?? "",
    userId: "",
  };
}

function Section({ title, children }: { title: string; children: React.ReactNode }) {
  return (
    <Card className="gap-4">
      <CardHeader>
        <CardTitle className="text-[15px]">{title}</CardTitle>
      </CardHeader>
      <CardContent className="grid gap-4 sm:grid-cols-2">{children}</CardContent>
    </Card>
  );
}

/** Create or edit an employee (NüHabit employees/insert and edit). */
export function EmployeeForm({
  profile,
  departments,
  positions,
  statuses,
  managers,
  users,
}: {
  profile?: EmployeeProfile;
  departments: Department[];
  positions: Position[];
  statuses: EmploymentStatusOption[];
  managers: EmployeeOption[];
  users: UserOption[];
}) {
  const router = useRouter();
  const id = profile?.employee.id ?? null;
  const [v, setV] = React.useState<EmployeeInput>(() => fromProfile(profile));
  const { errors, onFail } = useFieldErrors();
  const { pending, run } = useErpAction();
  const set = <K extends keyof EmployeeInput>(k: K) => (val: EmployeeInput[K]) => setV((s) => ({ ...s, [k]: val }));
  const text = (k: keyof EmployeeInput) => ({
    id: k,
    value: v[k],
    onChange: (e: React.ChangeEvent<HTMLInputElement | HTMLTextAreaElement>) => set(k)(e.target.value),
    "aria-invalid": errors?.[k] ? true : undefined,
  });
  const deptPositions = positions.filter((p) => p.isActive && (!v.departmentId || p.departmentId === v.departmentId));

  return (
    <form
      className="flex flex-col gap-4"
      onSubmit={(e) => {
        e.preventDefault();
        void run(() => saveEmployeeAction(id, v), {
          success: id ? th.employees.updated : th.employees.created,
          onFail,
          onOk: (res) => router.push(`/erp/hr/employees/${res.employee.id}`),
        });
      }}
    >
      <Section title={th.employees.sections.employment}>
        <Fld label={f.fullName} htmlFor="fullName" error={errors?.fullName}>
          <Input {...text("fullName")} autoFocus={!id} />
        </Fld>
        <Fld label={f.employeeNo} htmlFor="employeeNo" error={errors?.employeeNo} hint={f.employeeNoHint}>
          <Input {...text("employeeNo")} />
        </Fld>
        <Fld label={th.employees.email} htmlFor="email" error={errors?.email}>
          <Input type="email" {...text("email")} />
        </Fld>
        <Fld label={th.employees.phone} htmlFor="phone" error={errors?.phone}>
          <Input type="tel" inputMode="tel" {...text("phone")} />
        </Fld>
        <Fld label={th.department}>
          <SimpleSelect
            value={v.departmentId}
            onChange={(departmentId) => setV((s) => ({ ...s, departmentId, positionId: "" }))}
            emptyLabel={th.none}
            options={departments.filter((d) => d.isActive).map((d) => ({ value: d.id, label: d.name }))}
          />
        </Fld>
        <Fld label={th.position}>
          <SimpleSelect
            value={v.positionId}
            onChange={set("positionId")}
            emptyLabel={th.none}
            options={deptPositions.map((p) => ({ value: p.id, label: p.title }))}
          />
        </Fld>
        <Fld label={th.employees.manager}>
          <EmployeePicker
            value={v.managerId}
            onChange={set("managerId")}
            options={managers.filter((m) => m.id !== id)}
            allowEmpty={th.employees.noManager}
          />
        </Fld>
        <Fld label={f.employmentStatus}>
          <SimpleSelect
            value={v.employmentStatus}
            onChange={set("employmentStatus")}
            options={statuses.filter((s) => s.isActive).map((s) => ({ value: s.code, label: s.name }))}
          />
        </Fld>
        <Fld label={th.employees.joinDate} error={errors?.joinDate}>
          <DatePicker value={v.joinDate} onChange={set("joinDate")} invalid={Boolean(errors?.joinDate)} />
        </Fld>
        <Fld label={f.probationEndDate} error={errors?.probationEndDate}>
          <DatePicker value={v.probationEndDate} onChange={set("probationEndDate")} clearable />
        </Fld>
        <Fld label={f.endDate} error={errors?.endDate}>
          <DatePicker value={v.endDate} onChange={set("endDate")} clearable />
        </Fld>
      </Section>

      <Section title={th.employees.sections.personal}>
        <Fld label={f.birthPlace} htmlFor="birthPlace">
          <Input {...text("birthPlace")} />
        </Fld>
        <Fld label={f.birthDate} error={errors?.birthDate}>
          <DatePicker value={v.birthDate} onChange={set("birthDate")} clearable />
        </Fld>
        <Fld label={f.gender}>
          <SimpleSelect
            value={v.gender}
            onChange={set("gender")}
            emptyLabel="–"
            options={Object.entries(th.genders).map(([value, label]) => ({ value, label }))}
          />
        </Fld>
        <Fld label={f.maritalStatus}>
          <SimpleSelect
            value={v.maritalStatus}
            onChange={set("maritalStatus")}
            emptyLabel="–"
            options={Object.entries(th.maritalStatuses).map(([value, label]) => ({ value, label }))}
          />
        </Fld>
        <Fld label={f.religion}>
          <SimpleSelect
            value={v.religion}
            onChange={set("religion")}
            emptyLabel="–"
            options={th.religions.map((r) => ({ value: r, label: r }))}
          />
        </Fld>
        <Fld label={f.address} htmlFor="address" className="sm:col-span-2">
          <Textarea rows={2} {...text("address")} />
        </Fld>
      </Section>

      <Section title={th.employees.sections.identity}>
        <Fld label={f.nik} htmlFor="nik" error={errors?.nik}>
          <Input inputMode="numeric" maxLength={16} {...text("nik")} />
        </Fld>
        <Fld label={f.npwp} htmlFor="npwp">
          <Input {...text("npwp")} />
        </Fld>
        <Fld label={f.bpjsTkNo} htmlFor="bpjsTkNo">
          <Input {...text("bpjsTkNo")} />
        </Fld>
        <Fld label={f.bpjsKesNo} htmlFor="bpjsKesNo">
          <Input {...text("bpjsKesNo")} />
        </Fld>
      </Section>

      <Section title={th.employees.sections.bank}>
        <Fld label={f.bankName} htmlFor="bankName">
          <Input {...text("bankName")} />
        </Fld>
        <Fld label={f.bankAccountNo} htmlFor="bankAccountNo">
          <Input inputMode="numeric" {...text("bankAccountNo")} />
        </Fld>
        <Fld label={f.bankAccountName} htmlFor="bankAccountName" className="sm:col-span-2">
          <Input {...text("bankAccountName")} />
        </Fld>
      </Section>

      <Section title={th.employees.sections.emergency}>
        <Fld label={f.emergencyName} htmlFor="emergencyContactName">
          <Input {...text("emergencyContactName")} />
        </Fld>
        <Fld label={f.emergencyPhone} htmlFor="emergencyContactPhone">
          <Input type="tel" {...text("emergencyContactPhone")} />
        </Fld>
        <Fld label={f.emergencyRelation} htmlFor="emergencyContactRelation">
          <Input {...text("emergencyContactRelation")} />
        </Fld>
      </Section>

      {!id ? (
        <Section title={th.employees.sections.account}>
          <Fld label={f.userId} hint={th.employees.linkBody} className="sm:col-span-2">
            <SimpleSelect
              value={v.userId}
              onChange={set("userId")}
              emptyLabel={th.employees.notLinked}
              options={users.map((u) => ({ value: u.id, label: `${u.name} · ${u.email}` }))}
            />
          </Fld>
        </Section>
      ) : null}

      <div className="sticky bottom-[calc(env(safe-area-inset-bottom)+4.5rem)] z-10 flex justify-end gap-2 rounded-2xl bg-background/90 py-2 backdrop-blur md:bottom-2">
        <Button type="button" variant="outline" onClick={() => router.back()} disabled={pending}>
          {th.cancel}
        </Button>
        <Button type="submit" disabled={pending}>
          {pending ? <Loader2 className="size-4 animate-spin" /> : null}
          {th.save}
        </Button>
      </div>
    </form>
  );
}
