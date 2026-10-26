"use client";

import * as React from "react";
import type { z } from "zod";
import Link from "next/link";
import type { ColumnDef } from "@tanstack/react-table";
import { Copy, ExternalLink, MoreHorizontal, Pencil, Plus, Send, Trash2, XCircle } from "lucide-react";
import { Button } from "@/components/ui/button";
import {
  DropdownMenu,
  DropdownMenuContent,
  DropdownMenuItem,
  DropdownMenuSeparator,
  DropdownMenuTrigger,
} from "@/components/ui/dropdown-menu";
import { Input } from "@/components/ui/input";
import { Textarea } from "@/components/ui/textarea";
import { ConfirmAction, Fld, FormDialog, FormFooter, SimpleSelect, useFieldErrors } from "@/features/erp/hr/components/ui";
import { useErpAction } from "@/features/erp/shared/client";
import { ErpDataTable } from "@/features/erp/shared/data-table";
import { MoneyInput } from "@/features/erp/shared/money-input";
import { Money } from "@/features/erp/shared/money-display";
import { DatePicker } from "@/features/erp/shared/period-picker";
import { SearchableSelect, type SearchOption } from "@/features/erp/shared/searchable-select";
import { ErpStatusBadge } from "@/features/erp/shared/status-badge";
import { ErpEmpty } from "@/features/erp/shared/states";
import { ss } from "@/features/erp/shared/strings";
import { toast } from "@/lib/toast";
import { ResetFilters, SearchFilter, SelectFilter } from "@/features/erp/shared/url-filters";
import { formatDate, formatNumber } from "@/lib/format";
import { deleteJobAction, saveJobAction } from "../actions";
import { EMPLOYMENT_TYPES, JOB_STATUS, WORK_MODES, kinds, rs } from "../strings";
import type { jobSchema } from "../schemas";
import type { JobOpening } from "../types";

type JobOpeningInput = z.output<typeof jobSchema>;
type JobStatus = "draft" | "published" | "closed";

type Lookups = { departments: SearchOption[]; users: SearchOption[] };

const j = rs.jobs;
const STATUS_OPTIONS = Object.entries(JOB_STATUS).map(([value, label]) => ({ value, label }));

function toInput(job: JobOpening, status = job.status as JobStatus): JobOpeningInput {
  return {
    title: job.title,
    slug: job.slug,
    departmentId: job.departmentId ?? "",
    department: job.department,
    location: job.location,
    employmentType: job.employmentType,
    workMode: job.workMode,
    headcount: job.headcount,
    description: job.description,
    requirements: job.requirements,
    benefits: job.benefits,
    salaryMin: job.salaryMin,
    salaryMax: job.salaryMax,
    hiringManagerUserId: job.hiringManagerUserId ?? "",
    status,
    closingDate: job.closingDate ?? "",
  };
}

export function JobOpeningsView({ jobs, canWrite, filtered, ...lookups }: Lookups & { jobs: JobOpening[]; canWrite: boolean; filtered: boolean }) {
  const columns = React.useMemo<ColumnDef<JobOpening, unknown>[]>(
    () => [
      {
        accessorKey: "title",
        header: j.titleCol,
        cell: ({ row }) => (
          <div className="min-w-48">
            <Link href={`/erp/hr/recruitment/candidates?job=${row.original.id}`} className="font-medium hover:underline">
              {row.original.title}
            </Link>
            <div className="text-xs text-muted-foreground">
              {[row.original.department, row.original.location, row.original.employmentType, row.original.workMode].filter(Boolean).join(" · ")}
            </div>
          </div>
        ),
      },
      {
        id: "salary",
        header: j.salary,
        cell: ({ row }) =>
          row.original.salaryMin || row.original.salaryMax ? (
            <span className="text-xs whitespace-nowrap">
              <Money value={row.original.salaryMin} /> – <Money value={row.original.salaryMax} />
            </span>
          ) : (
            rs.none
          ),
      },
      {
        accessorKey: "candidateCount",
        header: j.candidates,
        cell: ({ row }) => (
          <span className="tabular">
            {formatNumber(row.original.candidateCount)} / {formatNumber(row.original.headcount)}
          </span>
        ),
      },
      { accessorKey: "hiredCount", header: j.hired, cell: ({ row }) => <span className="tabular">{row.original.hiredCount}</span> },
      {
        accessorKey: "closingDate",
        header: j.closing,
        cell: ({ row }) => <span className="whitespace-nowrap">{formatDate(row.original.closingDate)}</span>,
      },
      {
        accessorKey: "status",
        header: j.status,
        cell: ({ row }) => <ErpStatusBadge status={row.original.status} labels={JOB_STATUS} kinds={kinds.job} />,
      },
      {
        id: "actions",
        header: "",
        cell: ({ row }) => <JobActions job={row.original} canWrite={canWrite} {...lookups} />,
      },
    ],
    [canWrite, lookups],
  );

  if (!jobs.length && !filtered)
    return (
      <ErpEmpty
        title={j.empty}
        description={j.emptyDesc}
        action={canWrite ? <JobFormButton {...lookups} /> : undefined}
      />
    );

  return (
    <ErpDataTable
      columns={columns}
      data={jobs}
      exportName="lowongan"
      toolbar={
        <div className="flex flex-wrap items-center gap-2">
          <SearchFilter placeholder={j.search} />
          <SelectFilter param="status" options={STATUS_OPTIONS} label={j.statusFilter} allLabel={ss.allStatus} />
          <ResetFilters params={["q", "status"]} />
        </div>
      }
    />
  );
}

function JobActions({ job, canWrite, ...lookups }: Lookups & { job: JobOpening; canWrite: boolean }) {
  const { run, pending } = useErpAction();
  const [editing, setEditing] = React.useState(false);
  const publicPath = `/karier/${job.slug}`;
  const setStatus = (status: JobStatus, success: string) => run(() => saveJobAction(job.id, toInput(job, status)), { success });
  return (
    <div className="flex items-center justify-end gap-1">
      {job.status === "published" ? (
        <Button variant="ghost" size="icon-sm" asChild aria-label={j.publicPage}>
          <a href={publicPath} target="_blank" rel="noreferrer">
            <ExternalLink className="size-4" />
          </a>
        </Button>
      ) : null}
      {canWrite ? (
        <>
          <DropdownMenu>
            <DropdownMenuTrigger asChild>
              <Button variant="ghost" size="icon-sm" disabled={pending} aria-label={rs.edit}>
                <MoreHorizontal className="size-4" />
              </Button>
            </DropdownMenuTrigger>
            <DropdownMenuContent align="end">
              <DropdownMenuItem onSelect={() => setEditing(true)}>
                <Pencil className="size-4" /> {rs.edit}
              </DropdownMenuItem>
              {job.status !== "published" ? (
                <DropdownMenuItem onSelect={() => void setStatus("published", j.published)}>
                  <Send className="size-4" /> {j.publish}
                </DropdownMenuItem>
              ) : (
                <DropdownMenuItem onSelect={() => void setStatus("closed", j.closed)}>
                  <XCircle className="size-4" /> {j.close}
                </DropdownMenuItem>
              )}
              {job.status === "published" ? (
                <DropdownMenuItem
                  onSelect={() =>
                    void navigator.clipboard
                      .writeText(`${window.location.origin}${publicPath}`)
                      .then(() => toast.success(rs.copied))
                  }
                >
                  <Copy className="size-4" /> {rs.copy}
                </DropdownMenuItem>
              ) : null}
              <DropdownMenuSeparator />
              <ConfirmAction
                title={j.deleteTitle}
                description={j.deleteBody}
                confirmLabel={rs.delete}
                success={j.deleted}
                action={() => deleteJobAction(job.id)}
                trigger={
                  <DropdownMenuItem variant="destructive" onSelect={(e) => e.preventDefault()}>
                    <Trash2 className="size-4" /> {rs.delete}
                  </DropdownMenuItem>
                }
              />
            </DropdownMenuContent>
          </DropdownMenu>
          <FormDialog title={j.editTitle} wide open={editing} onOpenChange={setEditing}>
            {(close) => <JobForm job={job} close={close} {...lookups} />}
          </FormDialog>
        </>
      ) : null}
    </div>
  );
}

export function JobFormButton(lookups: Lookups) {
  return (
    <FormDialog
      title={j.new}
      wide
      trigger={
        <Button size="sm">
          <Plus className="size-4" /> {j.new}
        </Button>
      }
    >
      {(close) => <JobForm close={close} {...lookups} />}
    </FormDialog>
  );
}

const EMPTY: JobOpeningInput = {
  title: "",
  slug: "",
  departmentId: "",
  department: "",
  location: "",
  employmentType: "Full-time",
  workMode: "On-site",
  headcount: 1,
  description: "",
  requirements: "",
  benefits: "",
  salaryMin: "",
  salaryMax: "",
  hiringManagerUserId: "",
  status: "draft",
  closingDate: "",
};

function JobForm({ job, close, departments, users }: Lookups & { job?: JobOpening; close: () => void }) {
  const [v, setV] = React.useState<JobOpeningInput>(job ? toInput(job) : EMPTY);
  const { errors, onFail } = useFieldErrors();
  const { run, pending } = useErpAction();
  const set = <K extends keyof JobOpeningInput>(k: K, val: JobOpeningInput[K]) => setV((p) => ({ ...p, [k]: val }));
  const asOptions = (list: string[], current: string) =>
    [...new Set([...list, current].filter(Boolean))].map((x) => ({ value: x, label: x }));

  return (
    <form
      className="grid gap-4 sm:grid-cols-2"
      onSubmit={(e) => {
        e.preventDefault();
        const department = departments.find((d) => d.value === v.departmentId)?.label ?? v.department;
        void run(() => saveJobAction(job?.id ?? null, { ...v, department }), {
          success: job ? j.updated : j.created,
          onOk: close,
          onFail,
        });
      }}
    >
      <Fld label={j.titleCol} htmlFor="job-title" error={errors?.title} className="sm:col-span-2">
        <Input id="job-title" value={v.title} onChange={(e) => set("title", e.target.value)} autoFocus />
      </Fld>
      <Fld label={j.department} htmlFor="job-dept" error={errors?.departmentId}>
        <SearchableSelect
          id="job-dept"
          options={departments}
          value={v.departmentId ?? ""}
          onChange={(x) => set("departmentId", x)}
          placeholder={j.noDepartment}
          clearable
        />
      </Fld>
      <Fld label={j.location} htmlFor="job-loc" error={errors?.location}>
        <Input id="job-loc" value={v.location} onChange={(e) => set("location", e.target.value)} placeholder="Jakarta" />
      </Fld>
      <Fld label={j.type} htmlFor="job-type">
        <SimpleSelect id="job-type" value={v.employmentType} onChange={(x) => set("employmentType", x)} options={asOptions(EMPLOYMENT_TYPES, v.employmentType)} />
      </Fld>
      <Fld label={j.workMode} htmlFor="job-mode">
        <SimpleSelect id="job-mode" value={v.workMode} onChange={(x) => set("workMode", x)} options={asOptions(WORK_MODES, v.workMode)} />
      </Fld>
      <Fld label={j.headcount} htmlFor="job-hc" error={errors?.headcount}>
        <Input id="job-hc" type="number" min={1} value={v.headcount} onChange={(e) => set("headcount", Number(e.target.value))} />
      </Fld>
      <Fld label={j.closingDate} htmlFor="job-close" error={errors?.closingDate}>
        <DatePicker id="job-close" value={v.closingDate ?? ""} onChange={(x) => set("closingDate", x)} clearable />
      </Fld>
      <Fld label={j.salaryMin} htmlFor="job-smin" error={errors?.salaryMin} hint={j.salaryHint}>
        <MoneyInput id="job-smin" value={v.salaryMin} onChange={(x) => set("salaryMin", x)} />
      </Fld>
      <Fld label={j.salaryMax} htmlFor="job-smax" error={errors?.salaryMax}>
        <MoneyInput id="job-smax" value={v.salaryMax} onChange={(x) => set("salaryMax", x)} />
      </Fld>
      <Fld label={j.hiringManager} htmlFor="job-hm" hint={j.hiringManagerHint}>
        <SearchableSelect
          id="job-hm"
          options={users}
          value={v.hiringManagerUserId ?? ""}
          onChange={(x) => set("hiringManagerUserId", x)}
          placeholder={j.noManager}
          clearable
        />
      </Fld>
      <Fld label={j.status} htmlFor="job-status">
        <SimpleSelect id="job-status" value={v.status} onChange={(x) => set("status", x as JobStatus)} options={STATUS_OPTIONS} />
      </Fld>
      <Fld label={j.slug} htmlFor="job-slug" hint={j.slugHint} error={errors?.slug} className="sm:col-span-2">
        <Input id="job-slug" value={v.slug} onChange={(e) => set("slug", e.target.value)} placeholder="account-executive" />
      </Fld>
      <Fld label={j.description} htmlFor="job-desc" error={errors?.description} className="sm:col-span-2">
        <Textarea id="job-desc" rows={4} value={v.description} onChange={(e) => set("description", e.target.value)} />
      </Fld>
      <Fld label={j.requirements} htmlFor="job-req" error={errors?.requirements} className="sm:col-span-2">
        <Textarea id="job-req" rows={4} value={v.requirements} onChange={(e) => set("requirements", e.target.value)} />
      </Fld>
      <Fld label={j.benefits} htmlFor="job-ben" error={errors?.benefits} className="sm:col-span-2">
        <Textarea id="job-ben" rows={3} value={v.benefits} onChange={(e) => set("benefits", e.target.value)} />
      </Fld>
      <div className="sm:col-span-2">
        <FormFooter onCancel={close} pending={pending} />
      </div>
    </form>
  );
}
