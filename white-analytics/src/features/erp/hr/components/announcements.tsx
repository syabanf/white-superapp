"use client";

import * as React from "react";
import { Megaphone, Pencil, Pin, Plus, Trash2 } from "lucide-react";
import { Badge } from "@/components/ui/badge";
import { Button } from "@/components/ui/button";
import { Card } from "@/components/ui/card";
import { Checkbox } from "@/components/ui/checkbox";
import { Input } from "@/components/ui/input";
import { Label } from "@/components/ui/label";
import { Switch } from "@/components/ui/switch";
import { Textarea } from "@/components/ui/textarea";
import { useErpAction } from "@/features/erp/shared/client";
import { DatePicker } from "@/features/erp/shared/period-picker";
import { ErpEmpty } from "@/features/erp/shared/states";
import { ErpStatusBadge } from "@/features/erp/shared/status-badge";
import { SelectFilter } from "@/features/erp/shared/url-filters";
import { formatDate, formatRelative } from "@/lib/format";
import { cn } from "@/lib/utils";
import { deleteAnnouncementAction, saveAnnouncementAction } from "../actions";
import { kinds, th } from "../strings";
import type { Announcement, Department } from "../types";
import { ConfirmAction, Fld, FormDialog, FormFooter, SimpleSelect, useFieldErrors } from "./ui";

const n = th.announcements;

/** One announcement; `manage` adds edit and delete. */
function AnnouncementCard({
  item,
  departments,
  manage,
}: {
  item: Announcement;
  departments?: Department[];
  manage?: boolean;
}) {
  const deptNames = departments?.filter((d) => item.departmentIds.includes(d.id)).map((d) => d.name);
  return (
    <Card className={cn("gap-2 p-4 sm:p-5", item.isPinned && "ring-1 ring-brand/30")}>
      <div className="flex items-start gap-3">
        <div className="min-w-0 flex-1">
          <div className="flex flex-wrap items-center gap-2">
            {item.isPinned ? <Pin className="size-3.5 text-brand" aria-label={n.pinnedBadge} /> : null}
            <h3 className="font-semibold">{item.title}</h3>
            {!manage && !item.isRead ? <Badge className="px-1.5 py-0 text-[10px]">{n.unread}</Badge> : null}
            {manage ? <ErpStatusBadge status={item.status} labels={n.statuses} kinds={kinds.announcement} /> : null}
          </div>
          <p className="mt-0.5 text-xs text-muted-foreground">
            {formatRelative(item.publishAt ?? item.createdAt)}
            {item.createdByName ? ` · ${item.createdByName}` : ""}
            {manage ? ` · ${item.targetScope === "global" ? n.targets.global : deptNames?.join(", ")}` : ""}
            {manage ? ` · ${n.reads(item.readCount)}` : ""}
          </p>
        </div>
        {manage ? (
          <div className="flex shrink-0 gap-1">
            <FormDialog
              title={n.editAnnouncement}
              wide
              trigger={
                <Button size="icon-xs" variant="ghost" aria-label={th.edit}>
                  <Pencil />
                </Button>
              }
            >
              {(close) => <AnnouncementForm item={item} departments={departments ?? []} close={close} />}
            </FormDialog>
            <ConfirmAction
              title={th.confirmDelete}
              confirmLabel={th.delete}
              success={th.deleted}
              action={() => deleteAnnouncementAction(item.id)}
              trigger={
                <Button size="icon-xs" variant="ghost" aria-label={th.delete}>
                  <Trash2 />
                </Button>
              }
            />
          </div>
        ) : null}
      </div>
      <p className="text-sm whitespace-pre-line">{item.body}</p>
      {item.tags.length || item.expiresAt ? (
        <div className="flex flex-wrap items-center gap-1.5">
          {item.tags.map((t) => (
            <Badge key={t} variant="secondary" className="text-[11px]">
              #{t}
            </Badge>
          ))}
          {item.expiresAt ? (
            <span className="text-xs text-muted-foreground">
              {n.expiresAt} {formatDate(item.expiresAt)}
            </span>
          ) : null}
        </div>
      ) : null}
    </Card>
  );
}

export function AnnouncementsAdmin({
  items,
  departments,
  canWrite,
}: {
  items: Announcement[];
  departments: Department[];
  canWrite: boolean;
}) {
  const sorted = [...items].sort((a, b) => Number(b.isPinned) - Number(a.isPinned));
  return (
    <div className="flex flex-col gap-3">
      <div className="flex flex-wrap items-center gap-2">
        <SelectFilter
          param="status"
          label={th.status}
          allLabel={th.allStatus}
          options={Object.entries(n.statuses).map(([value, label]) => ({ value, label }))}
        />
        {canWrite ? (
          <div className="ml-auto">
            <FormDialog
              title={n.newAnnouncement}
              wide
              trigger={
                <Button size="sm">
                  <Plus className="size-4" /> {n.newAnnouncement}
                </Button>
              }
            >
              {(close) => <AnnouncementForm departments={departments} close={close} />}
            </FormDialog>
          </div>
        ) : null}
      </div>
      {sorted.length === 0 ? (
        <ErpEmpty icon={<Megaphone className="size-5" />} title={n.empty} />
      ) : (
        sorted.map((a) => <AnnouncementCard key={a.id} item={a} departments={departments} manage={canWrite} />)
      )}
    </div>
  );
}

/** ISO timestamp → "YYYY-MM-DD" in WIB, for the date picker. */
function toDay(ts: string | null): string {
  return ts ? new Intl.DateTimeFormat("en-CA", { timeZone: "Asia/Jakarta" }).format(new Date(ts)) : "";
}

/** "YYYY-MM-DD" → start of that day in WIB as RFC 3339. */
function fromDay(day: string): string | null {
  return day ? `${day}T00:00:00+07:00` : null;
}

function AnnouncementForm({
  item,
  departments,
  close,
}: {
  item?: Announcement;
  departments: Department[];
  close: () => void;
}) {
  const [v, setV] = React.useState({
    title: item?.title ?? "",
    body: item?.body ?? "",
    tags: item?.tags.join(", ") ?? "",
    status: (item?.status ?? "published") as "draft" | "published" | "archived",
    isPinned: item?.isPinned ?? false,
    targetScope: (item?.targetScope ?? "global") as "global" | "departments",
    departmentIds: item?.departmentIds ?? [],
    publishAt: toDay(item?.publishAt ?? null),
    expiresAt: toDay(item?.expiresAt ?? null),
  });
  const { errors, onFail } = useFieldErrors();
  const { pending, run } = useErpAction();
  return (
    <form
      className="flex flex-col gap-4"
      onSubmit={(e) => {
        e.preventDefault();
        void run(
          () =>
            saveAnnouncementAction(item?.id ?? null, {
              ...v,
              tags: v.tags
                .split(",")
                .map((t) => t.trim())
                .filter(Boolean),
              publishAt: fromDay(v.publishAt),
              expiresAt: fromDay(v.expiresAt),
            }),
          { success: th.saved, onOk: close, onFail },
        );
      }}
    >
      <Fld label={n.titleField} htmlFor="title" error={errors?.title}>
        <Input id="title" value={v.title} onChange={(e) => setV({ ...v, title: e.target.value })} />
      </Fld>
      <Fld label={n.body} htmlFor="body" error={errors?.body}>
        <Textarea id="body" rows={6} value={v.body} onChange={(e) => setV({ ...v, body: e.target.value })} />
      </Fld>
      <div className="grid gap-4 sm:grid-cols-2">
        <Fld label={n.tags} htmlFor="tags" hint={n.tagsHint}>
          <Input id="tags" value={v.tags} onChange={(e) => setV({ ...v, tags: e.target.value })} />
        </Fld>
        <Fld label={th.status}>
          <SimpleSelect
            value={v.status}
            onChange={(s) => setV({ ...v, status: s as typeof v.status })}
            options={Object.entries(n.statuses).map(([value, label]) => ({ value, label }))}
          />
        </Fld>
        <Fld label={n.publishAt}>
          <DatePicker value={v.publishAt} onChange={(publishAt) => setV({ ...v, publishAt })} clearable />
        </Fld>
        <Fld label={n.expiresAt}>
          <DatePicker value={v.expiresAt} onChange={(expiresAt) => setV({ ...v, expiresAt })} clearable />
        </Fld>
        <Fld label={n.target}>
          <SimpleSelect
            value={v.targetScope}
            onChange={(s) => setV({ ...v, targetScope: s as typeof v.targetScope })}
            options={Object.entries(n.targets).map(([value, label]) => ({ value, label }))}
          />
        </Fld>
        <div className="flex items-end gap-2 pb-2">
          <Switch id="pinned" checked={v.isPinned} onCheckedChange={(isPinned) => setV({ ...v, isPinned })} />
          <Label htmlFor="pinned">{n.pinned}</Label>
        </div>
      </div>
      {v.targetScope === "departments" ? (
        <Fld label={n.departments} error={errors?.departmentIds}>
          <div className="grid grid-cols-2 gap-2 sm:grid-cols-3">
            {departments.map((d) => (
              <label key={d.id} className="flex items-center gap-2 rounded-lg border px-3 py-2 text-sm">
                <Checkbox
                  checked={v.departmentIds.includes(d.id)}
                  onCheckedChange={(on) =>
                    setV({
                      ...v,
                      departmentIds: on ? [...v.departmentIds, d.id] : v.departmentIds.filter((x) => x !== d.id),
                    })
                  }
                />
                {d.name}
              </label>
            ))}
          </div>
        </Fld>
      ) : null}
      <FormFooter onCancel={close} pending={pending} />
    </form>
  );
}
