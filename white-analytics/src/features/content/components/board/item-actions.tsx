"use client";

import * as React from "react";
import { useRouter } from "next/navigation";
import { ArrowRight, CalendarClock, Loader2, Palette, Pencil, RefreshCw, Trash2 } from "lucide-react";
import { toast } from "@/lib/toast";
import { Button } from "@/components/ui/button";
import { Toolbar } from "@/components/dashboard/layout";
import { Input } from "@/components/ui/input";
import { Label } from "@/components/ui/label";
import { Textarea } from "@/components/ui/textarea";
import { Select, SelectContent, SelectItem, SelectTrigger, SelectValue } from "@/components/ui/select";
import { Dialog, DialogContent, DialogDescription, DialogFooter, DialogHeader, DialogTitle } from "@/components/ui/dialog";
import { fromZoned, toZoned } from "@/features/publishing/time";
import { deleteItem, moveItem, scheduleItem, syncItem } from "@/features/content/actions";
import { FORMATS, ITEM_STATUSES, itemInputFrom } from "@/features/content/lib";
import { c } from "@/features/content/strings";
import type { ContentFormat, Item, ItemStatus, Person, SizePreset } from "@/features/content/types";
import { ItemFormDialog } from "../item-form-dialog";
import { DesignTaskFormDialog } from "../design/task-form-dialog";

type Props = {
  slug: string;
  item: Item;
  allowedMoves: ItemStatus[];
  people: Person[];
  pillars: string[];
  presets: SizePreset[];
  timezone: string;
};

type DialogKind = "edit" | "delete" | "revision" | "schedule" | "design" | null;

/** Workflow bar on the item page; shown to members who may write. Moves come from the API. */
export function ItemActions({ slug, item, allowedMoves, people, pillars, presets, timezone }: Props) {
  const router = useRouter();
  const [pending, start] = React.useTransition();
  const [dialog, setDialog] = React.useState<DialogKind>(null);
  const [note, setNote] = React.useState("");
  const initialAt = item.publishAt ? toZoned(new Date(item.publishAt), timezone) : null;
  const [date, setDate] = React.useState(initialAt?.date ?? item.plannedDate);
  const [time, setTime] = React.useState(initialAt?.time ?? "10:00");
  const [format, setFormat] = React.useState<ContentFormat>(item.format);
  const locked = item.status === "SCHEDULED" || item.status === "PUBLISHED";
  const canSchedule = item.status === "APPROVED" || item.status === "FAILED";

  const run = (fn: () => Promise<{ ok: true } | { ok: false; error: string }>, success: string, after?: () => void) =>
    start(async () => {
      const res = await fn();
      if (!res.ok) {
        toast.error(res.error);
        return;
      }
      toast.success(success);
      setDialog(null);
      setNote("");
      if (after) after();
      else router.refresh();
    });

  const schedule = () =>
    start(async () => {
      const res = await scheduleItem({
        slug,
        itemId: item.id,
        publishAt: fromZoned(date, time, timezone).toISOString(),
        format,
        item: itemInputFrom(item),
      });
      if (!res.ok) {
        toast.error(res.error);
        return;
      }
      if (res.data.status === "FAILED") toast.warning(res.data.publishError ?? c.itemStatus.FAILED);
      else toast.success(c.scheduled);
      setDialog(null);
      router.refresh();
    });

  const sync = () =>
    start(async () => {
      const res = await syncItem({ slug, itemId: item.id });
      if (res.ok) {
        toast.success(`${c.synced}: ${c.itemStatus[res.data.status]}`);
        router.refresh();
      } else toast.error(res.error);
    });

  return (
    <>
      <Toolbar label={c.itemActions}>
        {allowedMoves.map((to) => (
          <Button
            key={to}
            variant={to === "REVISION" || ITEM_STATUSES.indexOf(to) < ITEM_STATUSES.indexOf(item.status) ? "outline" : "default"}
            disabled={pending}
            onClick={() => (to === "REVISION" ? setDialog("revision") : run(() => moveItem({ slug, itemId: item.id, to }), c.moved(c.itemStatus[to])))}
          >
            <ArrowRight /> {c.itemStatus[to]}
          </Button>
        ))}
        {canSchedule ? (
          <Button disabled={pending} onClick={() => setDialog("schedule")}>
            <CalendarClock /> {c.schedule}
          </Button>
        ) : null}
        {item.publishingPostId ? (
          <Button variant="outline" disabled={pending} onClick={sync}>
            <RefreshCw /> {c.sync}
          </Button>
        ) : null}
        {!locked ? (
          <Button variant="ghost" disabled={pending} onClick={() => setDialog("edit")}>
            <Pencil /> {c.edit}
          </Button>
        ) : null}
        <Button variant="ghost" disabled={pending} onClick={() => setDialog("design")}>
          <Palette /> {c.addDesignTask}
        </Button>
        {!locked ? (
          <Button variant="destructive" className="ml-auto" disabled={pending} onClick={() => setDialog("delete")}>
            <Trash2 /> {c.deleteItem}
          </Button>
        ) : null}
        {pending ? <Loader2 className="size-4 animate-spin text-muted-foreground" /> : null}
      </Toolbar>

      {dialog === "edit" ? (
        <ItemFormDialog
          open
          onOpenChange={(o) => !o && setDialog(null)}
          slug={slug}
          timezone={timezone}
          people={people}
          pillars={pillars}
          mode={{ kind: "edit", itemId: item.id, initial: itemInputFrom(item) }}
        />
      ) : null}

      {dialog === "design" ? (
        <DesignTaskFormDialog
          open
          onOpenChange={(o) => !o && setDialog(null)}
          slug={slug}
          people={people}
          presets={presets.filter((p) => p.format === item.format)}
          mode={{
            kind: "create",
            itemId: item.id,
            initial: {
              title: `Desain: ${item.title}`,
              brief: item.visualBrief,
              references: [],
              sizePreset: "",
              designerId: item.designerId ?? "",
              dueDate: item.designDue ?? "",
            },
          }}
        />
      ) : null}

      <Dialog open={dialog === "revision"} onOpenChange={(o) => !o && setDialog(null)}>
        <DialogContent>
          <DialogHeader>
            <DialogTitle>{c.revisionTitle}</DialogTitle>
            <DialogDescription>{c.revisionDesc}</DialogDescription>
          </DialogHeader>
          <Textarea value={note} onChange={(e) => setNote(e.target.value)} placeholder={c.notePlaceholder} rows={3} autoFocus />
          <DialogFooter>
            <Button variant="outline" onClick={() => setDialog(null)}>
              {c.cancel}
            </Button>
            <Button
             
              disabled={pending || !note.trim()}
              onClick={() => run(() => moveItem({ slug, itemId: item.id, to: "REVISION", note }), c.moved(c.itemStatus.REVISION))}
            >
              {c.revisionTitle}
            </Button>
          </DialogFooter>
        </DialogContent>
      </Dialog>

      <Dialog open={dialog === "schedule"} onOpenChange={(o) => !o && setDialog(null)}>
        <DialogContent className="sm:max-w-md">
          <DialogHeader>
            <DialogTitle>{c.scheduleTitle}</DialogTitle>
            <DialogDescription>{item.hasFinalDesign ? c.scheduleDesc : c.scheduleNeedsDesign}</DialogDescription>
          </DialogHeader>
          <div className="grid items-start gap-4 sm:grid-cols-2">
            <div className="space-y-1.5">
              <Label htmlFor="sch-date">{c.plannedDate}</Label>
              <Input id="sch-date" type="date" value={date} onChange={(e) => setDate(e.target.value)} />
            </div>
            <div className="space-y-1.5">
              <Label htmlFor="sch-time">{c.publishAt}</Label>
              <Input id="sch-time" type="time" step={300} value={time} onChange={(e) => setTime(e.target.value)} />
            </div>
            <div className="space-y-1.5 sm:col-span-2">
              <Label htmlFor="sch-format">{c.formatLabel}</Label>
              <Select value={format} onValueChange={(v) => setFormat(v as ContentFormat)}>
                <SelectTrigger id="sch-format" className="w-full">
                  <SelectValue />
                </SelectTrigger>
                <SelectContent>
                  {FORMATS.map((f) => (
                    <SelectItem key={f} value={f}>
                      {c.format[f]}
                    </SelectItem>
                  ))}
                </SelectContent>
              </Select>
            </div>
          </div>
          <DialogFooter>
            <Button variant="outline" onClick={() => setDialog(null)}>
              {c.cancel}
            </Button>
            <Button disabled={pending || !date || !time} onClick={schedule}>
              {pending ? <Loader2 className="size-3.5 animate-spin" /> : <CalendarClock className="size-3.5" />} {c.schedule}
            </Button>
          </DialogFooter>
        </DialogContent>
      </Dialog>

      <Dialog open={dialog === "delete"} onOpenChange={(o) => !o && setDialog(null)}>
        <DialogContent>
          <DialogHeader>
            <DialogTitle>{c.deleteItem}</DialogTitle>
            <DialogDescription>{c.deleteItemDesc}</DialogDescription>
          </DialogHeader>
          <DialogFooter>
            <Button variant="outline" onClick={() => setDialog(null)}>
              {c.cancel}
            </Button>
            <Button
             
              variant="destructive"
              disabled={pending}
              onClick={() => run(() => deleteItem({ slug, itemId: item.id }), c.itemDeleted, () => router.push(`/clients/${slug}/content/board`))}
            >
              {c.delete}
            </Button>
          </DialogFooter>
        </DialogContent>
      </Dialog>
    </>
  );
}
