"use client";

import * as React from "react";
import Link from "next/link";
import type { ColumnDef } from "@tanstack/react-table";
import { CalendarClock, FilePlus2, FileSignature, MoreHorizontal, UserX } from "lucide-react";
import { Button } from "@/components/ui/button";
import { Card, CardContent, CardHeader, CardTitle } from "@/components/ui/card";
import {
  DropdownMenu,
  DropdownMenuContent,
  DropdownMenuItem,
  DropdownMenuSeparator,
  DropdownMenuTrigger,
} from "@/components/ui/dropdown-menu";
import { Input } from "@/components/ui/input";
import { Textarea } from "@/components/ui/textarea";
import { useErpAction } from "@/features/erp/shared/client";
import { ErpDataTable } from "@/features/erp/shared/data-table";
import { Money } from "@/features/erp/shared/money-display";
import { formatRupiah } from "@/features/erp/shared/money";
import { MoneyInput } from "@/features/erp/shared/money-input";
import { DatePicker } from "@/features/erp/shared/period-picker";
import { ErpStatusBadge } from "@/features/erp/shared/status-badge";
import { ResetFilters, SearchFilter, SelectFilter } from "@/features/erp/shared/url-filters";
import { toast } from "@/lib/toast";
import { formatDate } from "@/lib/format";
import { cn } from "@/lib/utils";
import { contractActionAction, deleteContractAction, saveContractAction } from "../actions";
import { kinds, th } from "../strings";
import type { Contract, ExpiringContracts } from "../types";
import { EmployeePicker, Fld, FormDialog, FormFooter, SimpleSelect, useFieldErrors, type EmployeeOption } from "./ui";

const c = th.contracts;

/** Which actions a contract in this status allows (mirrors ContractService rules). */
function actionsFor(k: Contract): string[] {
  if (k.status === "draft") return ["activate"];
  if (k.status !== "active") return [];
  return k.contractType === "pkwt" ? ["renew", "convert", "end", "terminate"] : ["terminate"];
}

function DaysLeft({ days }: { days?: number }) {
  if (days == null) return <span className="text-muted-foreground">–</span>;
  return (
    <span className={cn("tabular", days < 0 ? "text-destructive" : days <= 30 ? "font-medium text-amber-600 dark:text-amber-400" : "")}>
      {c.daysLeftValue(days)}
    </span>
  );
}

export function ContractsTable({
  contracts,
  employees,
  canWrite,
  filters = true,
  defaultEmployeeId,
}: {
  contracts: Contract[];
  employees: EmployeeOption[];
  canWrite: boolean;
  filters?: boolean;
  defaultEmployeeId?: string;
}) {
  const [edit, setEdit] = React.useState<Contract | null>(null);
  const [act, setAct] = React.useState<{ contract: Contract; action: string } | null>(null);
  const { run } = useErpAction();

  const columns: ColumnDef<Contract, unknown>[] = [
    {
      accessorKey: "contractNumber",
      header: c.number,
      cell: ({ row }) => (
        <span className="min-w-0">
          <span className="block font-mono text-xs">{row.original.contractNumber}</span>
          <span className="text-xs text-muted-foreground">
            {c.sequence}
            {row.original.sequence}
          </span>
        </span>
      ),
    },
    {
      id: "employee",
      accessorFn: (r) => r.employee.fullName,
      header: th.employee,
      cell: ({ row }) => (
        <Link
          href={`/erp/hr/employees/${row.original.employee.id}?tab=contracts`}
          className="font-medium hover:underline"
          onClick={(e) => e.stopPropagation()}
        >
          {row.original.employee.fullName}
          <span className="block text-xs font-normal text-muted-foreground">{row.original.positionTitle || "–"}</span>
        </Link>
      ),
    },
    {
      accessorKey: "contractType",
      header: c.contractType,
      cell: ({ getValue }) => c.typesShort[getValue<string>()] ?? getValue<string>(),
    },
    {
      id: "period",
      header: th.period,
      cell: ({ row }) => (
        <span className="whitespace-nowrap tabular">
          {formatDate(row.original.startDate)} – {row.original.endDate ? formatDate(row.original.endDate) : "∞"}
        </span>
      ),
    },
    { id: "daysLeft", header: c.daysLeft, cell: ({ row }) => <DaysLeft days={row.original.daysLeft} /> },
    {
      accessorKey: "baseSalary",
      header: c.salary,
      meta: { align: "right" },
      cell: ({ getValue }) => <Money value={getValue<string>()} muteZero />,
    },
    {
      accessorKey: "status",
      header: th.status,
      cell: ({ getValue }) => <ErpStatusBadge status={getValue<string>()} labels={c.statuses} kinds={kinds.contract} />,
    },
    {
      id: "actions",
      header: "",
      cell: ({ row }) => {
        const k = row.original;
        const acts = actionsFor(k);
        if (!canWrite || (!acts.length && k.status !== "draft")) return null;
        return (
          <DropdownMenu>
            <DropdownMenuTrigger asChild>
              <Button size="icon-xs" variant="ghost" aria-label={th.actions}>
                <MoreHorizontal />
              </Button>
            </DropdownMenuTrigger>
            <DropdownMenuContent align="end">
              {k.status === "draft" ? <DropdownMenuItem onSelect={() => setEdit(k)}>{th.edit}</DropdownMenuItem> : null}
              {acts.map((a) => (
                <DropdownMenuItem key={a} onSelect={() => setAct({ contract: k, action: a })}>
                  {c.actions[a]}
                </DropdownMenuItem>
              ))}
              {k.status === "draft" ? (
                <>
                  <DropdownMenuSeparator />
                  <DropdownMenuItem
                    variant="destructive"
                    onSelect={() => void run(() => deleteContractAction(k.id), { success: th.deleted })}
                  >
                    {th.delete}
                  </DropdownMenuItem>
                </>
              ) : null}
            </DropdownMenuContent>
          </DropdownMenu>
        );
      },
    },
  ];

  const newButton = canWrite ? (
    <FormDialog
      title={c.newContract}
      wide
      trigger={
        <Button size="sm">
          <FilePlus2 className="size-4" /> {c.newContract}
        </Button>
      }
    >
      {(close) => <ContractForm employees={employees} defaultEmployeeId={defaultEmployeeId} close={close} />}
    </FormDialog>
  ) : null;

  return (
    <>
      <ErpDataTable
        columns={columns}
        data={contracts}
        getRowId={(r) => r.id}
        toolbar={
          filters ? (
            <>
              <SearchFilter placeholder={c.search} />
              <SelectFilter
                param="status"
                label={th.status}
                allLabel={c.statuses.active}
                options={[{ value: "all", label: th.allStatus }, ...Object.entries(c.statuses).filter(([k]) => k !== "active").map(([value, label]) => ({ value, label }))]}
              />
              <SelectFilter
                param="type"
                label={c.contractType}
                allLabel={th.allTypes}
                options={Object.entries(c.typesShort).map(([value, label]) => ({ value, label }))}
              />
              <ResetFilters params={["q", "status", "type"]} />
              <div className="ml-auto">{newButton}</div>
            </>
          ) : (
            <div className="ml-auto">{newButton}</div>
          )
        }
      />
      <FormDialog title={c.editContract} wide open={edit !== null} onOpenChange={(o) => !o && setEdit(null)}>
        {(close) => edit && <ContractForm contract={edit} employees={employees} close={close} />}
      </FormDialog>
      <FormDialog
        title={act ? `${c.actions[act.action]} · ${act.contract.contractNumber}` : ""}
        description={act ? c.actionBody[act.action] : undefined}
        open={act !== null}
        onOpenChange={(o) => !o && setAct(null)}
      >
        {(close) => act && <ContractActionForm contract={act.contract} action={act.action} close={close} />}
      </FormDialog>
    </>
  );
}

function ContractForm({
  contract,
  employees,
  defaultEmployeeId,
  close,
}: {
  contract?: Contract;
  employees: EmployeeOption[];
  defaultEmployeeId?: string;
  close: () => void;
}) {
  const [v, setV] = React.useState({
    employeeId: contract?.employee.id ?? defaultEmployeeId ?? "",
    contractType: (contract?.contractType ?? "pkwt") as "pkwt" | "pkwtt",
    startDate: contract?.startDate ?? "",
    endDate: contract?.endDate ?? "",
    probationEndDate: contract?.probationEndDate ?? "",
    positionTitle: contract?.positionTitle ?? "",
    departmentName: contract?.departmentName ?? "",
    workLocation: contract?.workLocation ?? "",
    baseSalary: contract?.baseSalary ?? "",
    notes: contract?.notes ?? "",
  });
  const { errors, onFail } = useFieldErrors();
  const { pending, run } = useErpAction();
  return (
    <form
      className="flex flex-col gap-4"
      onSubmit={(e) => {
        e.preventDefault();
        void run(() => saveContractAction(contract?.id ?? null, v), {
          success: contract ? c.updated : c.created,
          onOk: close,
          onFail,
        });
      }}
    >
      <div className="grid gap-4 sm:grid-cols-2">
        <Fld label={th.employee} error={errors?.employeeId}>
          <EmployeePicker
            value={v.employeeId}
            onChange={(employeeId) => setV({ ...v, employeeId })}
            options={employees}
            invalid={Boolean(errors?.employeeId)}
          />
        </Fld>
        <Fld label={c.contractType}>
          <SimpleSelect
            value={v.contractType}
            onChange={(t) => setV({ ...v, contractType: t as "pkwt" | "pkwtt", endDate: t === "pkwtt" ? "" : v.endDate })}
            options={Object.entries(c.types).map(([value, label]) => ({ value, label }))}
          />
        </Fld>
        <Fld label={c.start} error={errors?.startDate}>
          <DatePicker value={v.startDate} onChange={(startDate) => setV({ ...v, startDate })} />
        </Fld>
        <Fld label={c.end} error={errors?.endDate}>
          <DatePicker
            value={v.endDate}
            onChange={(endDate) => setV({ ...v, endDate })}
            disabled={v.contractType === "pkwtt"}
            clearable
          />
        </Fld>
        <Fld label={c.probationEnd} error={errors?.probationEndDate}>
          <DatePicker value={v.probationEndDate} onChange={(probationEndDate) => setV({ ...v, probationEndDate })} clearable />
        </Fld>
        <Fld label={c.salary} error={errors?.baseSalary}>
          <MoneyInput value={v.baseSalary} onChange={(baseSalary) => setV({ ...v, baseSalary })} />
        </Fld>
        <Fld label={c.positionTitle} htmlFor="positionTitle">
          <Input id="positionTitle" value={v.positionTitle} onChange={(e) => setV({ ...v, positionTitle: e.target.value })} />
        </Fld>
        <Fld label={c.departmentName} htmlFor="departmentName">
          <Input
            id="departmentName"
            value={v.departmentName}
            onChange={(e) => setV({ ...v, departmentName: e.target.value })}
          />
        </Fld>
        <Fld label={c.location} htmlFor="workLocation" className="sm:col-span-2">
          <Input id="workLocation" value={v.workLocation} onChange={(e) => setV({ ...v, workLocation: e.target.value })} />
        </Fld>
        <Fld label={th.notes} htmlFor="notes" className="sm:col-span-2">
          <Textarea id="notes" rows={2} value={v.notes} onChange={(e) => setV({ ...v, notes: e.target.value })} />
        </Fld>
      </div>
      <FormFooter onCancel={close} pending={pending} />
    </form>
  );
}

function ContractActionForm({ contract, action, close }: { contract: Contract; action: string; close: () => void }) {
  const [date, setDate] = React.useState("");
  const [reason, setReason] = React.useState("");
  const { pending, run } = useErpAction();
  const needsDate = action === "activate" || action === "end" || action === "renew";
  const dateLabel = action === "activate" ? c.signedAt : action === "renew" ? c.end : c.actionDate;
  return (
    <form
      className="flex flex-col gap-4"
      onSubmit={(e) => {
        e.preventDefault();
        void run(
          () =>
            contractActionAction(contract.id, {
              action,
              ...(action === "activate" ? { signedAt: date } : { endDate: date }),
              reason,
            }),
          {
            onOk: (res) => {
              toast.success(res.message || th.saved, {
                description: Number(res.compensationAmount) > 0 ? c.compensation(formatRupiah(res.compensationAmount)) : undefined,
              });
              close();
            },
          },
        );
      }}
    >
      {needsDate ? (
        <Fld label={dateLabel}>
          <DatePicker value={date} onChange={setDate} clearable={action !== "renew"} />
        </Fld>
      ) : null}
      {action === "terminate" ? (
        <Fld label={th.reason} htmlFor="reason">
          <Textarea id="reason" rows={3} value={reason} onChange={(e) => setReason(e.target.value)} />
        </Fld>
      ) : null}
      <FormFooter
        onCancel={close}
        pending={pending}
        submitLabel={c.actions[action]}
        destructive={action === "terminate"}
      />
    </form>
  );
}

/** Contracts ending soon, probations ending, and active staff with no contract. */
export function ExpiringPanel({ data }: { data: ExpiringContracts }) {
  return (
    <div className="grid gap-4 lg:grid-cols-3">
      <ExpiringCard title={c.expiringContracts} count={data.contracts.length}>
        {data.contracts.map((k) => (
          <ExpiringRow
            key={k.id}
            icon={<FileSignature className="size-4" />}
            employee={k.employee}
            label={`${k.contractNumber} · ${c.typesShort[k.contractType] ?? k.contractType}`}
            date={k.endDate}
            days={k.daysLeft}
          />
        ))}
      </ExpiringCard>
      <ExpiringCard title={c.probations} count={data.probations.length}>
        {data.probations.map((p) => (
          <ExpiringRow
            key={p.employee.id}
            icon={<CalendarClock className="size-4" />}
            employee={p.employee}
            label={p.employee.departmentName}
            date={p.probationEndDate}
            days={p.daysLeft}
          />
        ))}
      </ExpiringCard>
      <ExpiringCard title={c.withoutContract} count={data.withoutContract.length}>
        {data.withoutContract.map((e) => (
          <ExpiringRow key={e.id} icon={<UserX className="size-4" />} employee={e} label={e.employeeNo} />
        ))}
      </ExpiringCard>
    </div>
  );
}

function ExpiringCard({ title, count, children }: { title: string; count: number; children: React.ReactNode }) {
  return (
    <Card className="gap-2 pb-2">
      <CardHeader className="flex flex-row items-center justify-between">
        <CardTitle className="text-[15px]">{title}</CardTitle>
        <span className="text-sm font-semibold tabular">{count}</span>
      </CardHeader>
      <CardContent className="px-2">
        {count === 0 ? (
          <p className="px-3 py-4 text-sm text-muted-foreground">{c.noneExpiring}</p>
        ) : (
          <ul className="flex flex-col">{children}</ul>
        )}
      </CardContent>
    </Card>
  );
}

function ExpiringRow({
  icon,
  employee,
  label,
  date,
  days,
}: {
  icon: React.ReactNode;
  employee: { id: string; fullName: string };
  label: string;
  date?: string;
  days?: number;
}) {
  return (
    <li>
      <Link
        href={`/erp/hr/employees/${employee.id}?tab=contracts`}
        className="flex items-center gap-3 rounded-lg px-3 py-2 hover:bg-accent/50"
      >
        <span className="text-muted-foreground">{icon}</span>
        <span className="min-w-0 flex-1">
          <span className="block truncate text-sm font-medium">{employee.fullName}</span>
          <span className="block truncate text-xs text-muted-foreground">
            {label}
            {date ? ` · ${formatDate(date)}` : ""}
          </span>
        </span>
        {days != null ? <DaysLeft days={days} /> : null}
      </Link>
    </li>
  );
}
