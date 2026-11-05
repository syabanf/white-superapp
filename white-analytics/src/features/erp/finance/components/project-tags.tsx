"use client";

import * as React from "react";
import type { ColumnDef } from "@tanstack/react-table";
import { Pencil, Plus, Trash2 } from "lucide-react";
import { Button } from "@/components/ui/button";
import { Input } from "@/components/ui/input";
import { ConfirmAction, Fld, FormDialog, FormFooter } from "@/features/erp/sales/components/ui";
import { useErpAction } from "@/features/erp/shared/client";
import { ErpDataTable } from "@/features/erp/shared/data-table";
import { ResetFilters, SearchFilter, SearchSelectFilter, SelectFilter } from "@/features/erp/shared/url-filters";
import { deleteProjectTagAction, saveProjectTagAction } from "../actions";
import { fs } from "../strings";
import type { ProjectTag } from "../types";

const s = fs.projects;

function TagForm({ tag, close }: { tag?: ProjectTag; close: () => void }) {
  const [v, setV] = React.useState<ProjectTag>(tag ?? { id: "", name: "", clientName: "", dealId: "", isActive: true, notes: "" });
  const [errors, setErrors] = React.useState<Record<string, string[]>>();
  const { pending, run } = useErpAction();
  return (
    <form
      className="grid gap-4"
      onSubmit={(e) => {
        e.preventDefault();
        void run(() => saveProjectTagAction(v), { success: s.saved, onOk: close, onFail: (r) => setErrors(r.fieldErrors) });
      }}
    >
      <Fld label={`${s.name} *`} htmlFor="tag-name" error={errors?.name}>
        <Input id="tag-name" value={v.name} onChange={(e) => setV((p) => ({ ...p, name: e.target.value }))} />
      </Fld>
      <Fld label={s.client} htmlFor="tag-client" hint={s.clientHint}>
        <Input id="tag-client" value={v.clientName} onChange={(e) => setV((p) => ({ ...p, clientName: e.target.value }))} />
      </Fld>
      <Fld label={s.notes} htmlFor="tag-notes">
        <Input id="tag-notes" value={v.notes} onChange={(e) => setV((p) => ({ ...p, notes: e.target.value }))} />
      </Fld>
      <label className="flex items-center gap-2 text-sm">
        <input type="checkbox" checked={v.isActive} onChange={(e) => setV((p) => ({ ...p, isActive: e.target.checked }))} /> {s.active}
      </label>
      <FormFooter onCancel={close} pending={pending} />
    </form>
  );
}

export function TagDialog({ tag }: { tag?: ProjectTag }) {
  const title = tag ? s.edit : s.new;
  return (
    <FormDialog
      title={title}
      trigger={
        tag ? (
          <Button size="icon-xs" variant="ghost" aria-label={title}>
            <Pencil className="size-3.5" />
          </Button>
        ) : (
          <Button size="sm">
            <Plus className="size-4" /> {title}
          </Button>
        )
      }
    >
      {(close) => <TagForm tag={tag} close={close} />}
    </FormDialog>
  );
}

export function ProjectTagsTable({ tags, clients, canWrite }: { tags: ProjectTag[]; clients: string[]; canWrite: boolean }) {
  const columns: ColumnDef<ProjectTag, unknown>[] = [
    {
      accessorKey: "name",
      header: s.name,
      cell: ({ row }) => (
        <div className="min-w-0">
          <div className="font-medium">{row.original.name}</div>
          {row.original.isActive ? null : <div className="text-xs text-muted-foreground">{s.inactive}</div>}
        </div>
      ),
    },
    { accessorKey: "clientName", header: s.client, cell: ({ row }) => row.original.clientName || "–" },
    { id: "source", header: s.source, cell: ({ row }) => (row.original.dealId ? s.fromDeal : s.manual) },
    { accessorKey: "notes", header: s.notes },
    ...(canWrite
      ? [
          {
            id: "actions",
            header: "",
            enableSorting: false,
            cell: ({ row }: { row: { original: ProjectTag } }) => (
              <span className="flex justify-end gap-0.5">
                <TagDialog tag={row.original} />
                <ConfirmAction
                  title={s.remove}
                  description={s.removeHint}
                  action={() => deleteProjectTagAction(row.original.id)}
                  trigger={
                    <Button size="icon-xs" variant="ghost" aria-label={s.remove} className="text-destructive">
                      <Trash2 className="size-3.5" />
                    </Button>
                  }
                />
              </span>
            ),
          } satisfies ColumnDef<ProjectTag, unknown>,
        ]
      : []),
  ];
  return (
    <ErpDataTable
      columns={columns}
      data={tags}
      getRowId={(t) => t.id}
      emptyMessage={s.empty}
      exportName="proyek"
      toolbar={
        <>
          <SearchFilter placeholder={s.search} />
          {clients.length ? <SearchSelectFilter param="client" placeholder={s.allClients} options={clients.map((c) => ({ value: c, label: c }))} className="w-56" /> : null}
          <SelectFilter param="status" label={s.status} allLabel={s.allStatus} options={Object.entries(s.statuses).map(([value, label]) => ({ value, label }))} />
          <ResetFilters params={["q", "client", "status"]} />
        </>
      }
    />
  );
}
